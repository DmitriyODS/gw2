package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// ListRegistries — реестры компании с их полями (батч-загрузка без N+1).
func (s *Service) ListRegistries(ctx context.Context, companyID int64) ([]*domain.Registry, error) {
	regs, err := s.repo.ListRegistries(ctx, companyID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(regs))
	for i, r := range regs {
		ids[i] = r.ID
	}
	byReg, err := s.repo.FieldsByRegistries(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range regs {
		fields := byReg[r.ID]
		if fields == nil {
			fields = []domain.Field{}
		}
		r.Fields = fields
	}
	return regs, nil
}

// GetRegistry — один реестр компании с полями.
func (s *Service) GetRegistry(ctx context.Context, companyID, id int64) (*domain.Registry, error) {
	reg, err := s.requireRegistry(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	fields, err := s.repo.ListFields(ctx, id)
	if err != nil {
		return nil, err
	}
	reg.Fields = fields
	return reg, nil
}

// CreateRegistry — новый реестр (структура полей задаётся отдельно).
func (s *Service) CreateRegistry(ctx context.Context, companyID, userID int64, name string) (*domain.Registry, error) {
	pos, err := s.repo.NextRegistryPosition(ctx, companyID)
	if err != nil {
		return nil, err
	}
	reg := &domain.Registry{CompanyID: companyID, Name: name, Position: pos, CreatedBy: &userID}
	if err := s.repo.CreateRegistry(ctx, reg); err != nil {
		return nil, err
	}
	reg.Fields = []domain.Field{}
	s.bus.Publish(ctx, "registry:created", []string{roomAll}, registryPayload(reg))
	return reg, nil
}

// UpdateRegistry — переименование (позиция не меняется).
func (s *Service) UpdateRegistry(ctx context.Context, companyID, id int64, name string) (*domain.Registry, error) {
	reg, err := s.requireRegistry(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateRegistry(ctx, id, name, reg.Position); err != nil {
		return nil, err
	}
	reg.Name = name
	s.bus.Publish(ctx, "registry:updated", []string{roomAll}, registryPayload(reg))
	return reg, nil
}

func (s *Service) DeleteRegistry(ctx context.Context, companyID, id int64) error {
	if _, err := s.requireRegistry(ctx, companyID, id); err != nil {
		return err
	}
	if err := s.repo.DeleteRegistry(ctx, id); err != nil {
		return err
	}
	s.bus.Publish(ctx, "registry:deleted", []string{roomAll}, map[string]any{
		"id": id, "company_id": companyID,
	})
	return nil
}

// ReplaceFields — полная замена набора полей. Отключённые (удалённые) поля
// вычищаются из данных всех записей с пересчётом search_text.
func (s *Service) ReplaceFields(ctx context.Context, companyID, id int64, fields []domain.Field) (*domain.Registry, error) {
	reg, err := s.requireRegistry(ctx, companyID, id)
	if err != nil {
		return nil, err
	}
	for i := range fields {
		fields[i].RegistryID = id
		fields[i].Normalize()
	}
	removed, err := s.repo.ReplaceFields(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		if err := s.stripRemovedFields(ctx, id, fields, removed); err != nil {
			return nil, err
		}
	}
	reg.Fields = fields
	s.bus.Publish(ctx, "registry:updated", []string{roomAll}, registryPayload(reg))
	return reg, nil
}

// stripRemovedFields — удалить значения отключённых полей из всех записей и
// пересчитать search_text по актуальному набору полей.
func (s *Service) stripRemovedFields(ctx context.Context, registryID int64, fields []domain.Field, removed []int64) error {
	records, err := s.repo.AllRecords(ctx, registryID)
	if err != nil {
		return err
	}
	var orphans []string
	for _, rec := range records {
		changed := false
		for _, fid := range removed {
			key := domain.FieldID(fid)
			if v, ok := rec.Data[key]; ok {
				if p := fileValuePath(v); p != "" {
					orphans = append(orphans, p)
				}
				delete(rec.Data, key)
				changed = true
			}
		}
		if !changed {
			continue
		}
		if err := s.repo.UpdateRecord(ctx, rec.ID, rec.Data, buildSearchText(fields, rec.Data)); err != nil {
			return err
		}
	}
	if len(orphans) > 0 {
		s.files.Remove(orphans)
	}
	return nil
}

func registryPayload(r *domain.Registry) map[string]any {
	return map[string]any{
		"id": r.ID, "company_id": r.CompanyID, "name": r.Name,
		"position": r.Position, "fields": r.Fields,
	}
}
