package service

import (
	"context"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

// ListRegistries — реестры выбранной области с их полями (батч-загрузка без N+1).
func (s *Service) ListRegistries(ctx context.Context, userID int64, scope string) ([]*domain.Registry, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	regs, err := s.repo.ListRegistries(ctx, a.UserID, domain.NormalizeScope(scope))
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

// TileSummary — сводка раздела для живой плитки «Пуска».
func (s *Service) TileSummary(ctx context.Context, userID int64) (*domain.RegistriesSummary, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.RegistriesSummary(ctx, a.UserID, 3)
}

// GetRegistry — один доступный реестр с полями.
func (s *Service) GetRegistry(ctx context.Context, userID, id int64) (*domain.Registry, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, id, domain.AccessView)
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

// CreateRegistry — новый реестр (структура полей задаётся отдельно) в личном
// пространстве (companyID == nil) либо в команде, где автор состоит.
func (s *Service) CreateRegistry(ctx context.Context, userID int64, companyID *int64, name string, accounting bool) (*domain.Registry, error) {
	if err := s.ensureLimit(ctx, userID); err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, userID, companyID); err != nil {
		return nil, err
	}
	pos, err := s.repo.NextRegistryPosition(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg := &domain.Registry{
		OwnerID: userID, CompanyID: companyID, TeamAccess: domain.AccessEdit, Name: name,
		Position: pos, Accounting: accounting, CreatedBy: &userID,
	}
	if err := s.repo.CreateRegistry(ctx, reg); err != nil {
		return nil, err
	}
	reg.Fields = []domain.Field{}
	reg.MyAccess = domain.AccessOwner
	s.publish(ctx, reg.ID, "registry:created", registryPayload(reg))
	return reg, nil
}

// RegistryPatch — правка реестра. Поле-источник подразделов и учётный режим
// меняются, только если пришёл соответствующий флаг: без него «ключа нет» не
// отличить от «выключить», и обычное переименование сбрасывало бы настройку.
type RegistryPatch struct {
	Name            string
	SectionFieldID  *int64
	SectionFieldSet bool
	Accounting      bool
	AccountingSet   bool
}

// UpdateRegistry — название, подразделы и учётный режим (позиция не меняется).
// Требует уровня admin: это правка самой структуры реестра.
func (s *Service) UpdateRegistry(ctx context.Context, userID, id int64, p RegistryPatch) (*domain.Registry, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, id, domain.AccessAdmin)
	if err != nil {
		return nil, err
	}
	return s.updateRegistryIn(ctx, reg, p)
}

// updateRegistryIn — правка уже проверенного реестра (участник с уровнем admin
// либо ссылка того же уровня).
func (s *Service) updateRegistryIn(ctx context.Context, reg *domain.Registry, p RegistryPatch) (*domain.Registry, error) {
	id := reg.ID
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = reg.Name
	}
	sectionFieldID, accounting := reg.SectionFieldID, reg.Accounting
	if p.SectionFieldSet {
		sectionFieldID = p.SectionFieldID
	}
	if p.AccountingSet {
		accounting = p.Accounting
	}

	/* Поля читаем ВСЕГДА, а не только ради проверки подраздела: ответ и событие
	   несут снимок реестра целиком, и снимок без полей клиент принимал за
	   «полей больше нет» — после переименования из таблицы пропадали все
	   колонки, кроме служебной «Создано». */
	fields, err := s.repo.ListFields(ctx, id)
	if err != nil {
		return nil, err
	}
	if sectionFieldID != nil {
		field := findField(fields, *sectionFieldID)
		if field == nil || field.Type != domain.FieldSelect {
			return nil, domain.ErrSectionFieldInvalid
		}
	}
	if err := s.repo.UpdateRegistry(ctx, id, name, reg.Position, sectionFieldID, accounting); err != nil {
		return nil, err
	}
	reg.Name, reg.SectionFieldID, reg.Accounting = name, sectionFieldID, accounting
	reg.Fields = fields
	s.publish(ctx, id, "registry:updated", registryPayload(reg))
	return reg, nil
}

// DeleteRegistry — только владелец (у реестра команды — автор или
// администратор команды): отдать реестр вместе со всеми записями — не то
// действие, которое доверяют приглашённому администратору.
func (s *Service) DeleteRegistry(ctx context.Context, userID, id int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	reg, err := s.require(ctx, a, id, domain.AccessOwner)
	if err != nil {
		return err
	}
	// Аудиторию узнаём ДО удаления: после него шар уже нет и звать некого.
	rooms := s.audience(ctx, id)
	records, err := s.repo.AllRecords(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRegistry(ctx, id); err != nil {
		return err
	}
	s.removeRecordFiles(ctx, reg, records...)
	s.bus.Publish(ctx, "registry:deleted", rooms, map[string]any{"id": id})
	return nil
}

// requireMember — положить вещь в команду может только её участник.
func (s *Service) requireMember(ctx context.Context, userID int64, companyID *int64) error {
	if companyID == nil {
		return nil
	}
	role, err := s.users.TeamRole(ctx, userID, *companyID)
	if err != nil {
		return err
	}
	if !role.Member {
		return domain.ErrNotTeamMember
	}
	return nil
}

/*
MoveRegistry — сменить пространство реестра и уровень участников команды.

	Распоряжается этим «владелец»: хозяин личного реестра, у реестра команды —
	автор и администраторы. Забирая реестр к себе, человек становится его
	хозяином. Учёт файлов переезжает ДО реестра: за личный платит хозяин, за
	командный — создатель команды, и сверка «Хранилища» стёрла бы файлы,
	оставшиеся в журнале прежнего плательщика. Не переехал учёт — не переезжает
	и реестр. Кто доступ потерял, получает событие удаления — иначе реестр висел
	бы у него в списке до перезагрузки.
*/
func (s *Service) MoveRegistry(ctx context.Context, userID, id int64, companyID *int64, teamAccess string) (*domain.Registry, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, id, domain.AccessOwner)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, userID, companyID); err != nil {
		return nil, err
	}
	ownerID := reg.OwnerID
	if companyID == nil {
		ownerID = userID
	}
	access := domain.NormalizeTeamAccess(teamAccess)
	if teamAccess == "" {
		access = reg.TeamAccess
	}

	before := s.audience(ctx, id)
	oldUser, oldCompany := quotaScope(reg)
	next := *reg
	next.OwnerID, next.CompanyID, next.TeamAccess = ownerID, companyID, access
	if newUser, newCompany := quotaScope(&next); newUser != oldUser || newCompany != oldCompany {
		if err := s.moveRecordFiles(ctx, &next); err != nil {
			return nil, domain.ErrMoveFailed
		}
	}
	if err := s.repo.MoveRegistry(ctx, id, ownerID, companyID, access); err != nil {
		return nil, err
	}
	reg = &next

	fields, err := s.repo.ListFields(ctx, id)
	if err != nil {
		return nil, err
	}
	reg.Fields = fields
	after := s.audience(ctx, id)
	if gone := missing(before, after); len(gone) > 0 {
		s.bus.Publish(ctx, "registry:deleted", gone, map[string]any{"id": id})
	}
	if len(after) > 0 {
		s.bus.Publish(ctx, "registry:updated", after, registryPayload(reg))
	}
	if reg.MyAccess, err = s.repo.AccessOf(ctx, id, userID); err != nil {
		return nil, err
	}
	return reg, nil
}

// moveRecordFiles — переписать файлы записей на плательщика реестра reg.
func (s *Service) moveRecordFiles(ctx context.Context, reg *domain.Registry) error {
	recs, err := s.repo.AllRecords(ctx, reg.ID)
	if err != nil {
		return err
	}
	var paths []string
	for _, rec := range recs {
		for _, v := range rec.Data {
			paths = append(paths, filePaths(v)...)
		}
	}
	userID, companyID := quotaScope(reg)
	return s.files.MoveFor(ctx, userID, companyID, paths)
}

// missing — комнаты, которые были в before и пропали из after.
func missing(before, after []string) []string {
	kept := make(map[string]bool, len(after))
	for _, r := range after {
		kept[r] = true
	}
	out := []string{}
	for _, r := range before {
		if !kept[r] {
			out = append(out, r)
		}
	}
	return out
}

// ReplaceFields — полная замена набора полей. Отключённые (удалённые) поля
// вычищаются из данных всех записей с пересчётом search_text.
func (s *Service) ReplaceFields(ctx context.Context, userID, id int64, fields []domain.Field) (*domain.Registry, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	reg, err := s.require(ctx, a, id, domain.AccessAdmin)
	if err != nil {
		return nil, err
	}
	return s.replaceFieldsIn(ctx, reg, fields)
}

// replaceFieldsIn — замена набора полей уже проверенного реестра.
func (s *Service) replaceFieldsIn(ctx context.Context, reg *domain.Registry, fields []domain.Field) (*domain.Registry, error) {
	id := reg.ID
	for i := range fields {
		fields[i].RegistryID = id
		fields[i].Normalize()
	}
	removed, err := s.repo.ReplaceFields(ctx, id, fields)
	if err != nil {
		return nil, err
	}
	if len(removed) > 0 {
		if err := s.stripRemovedFields(ctx, reg, fields, removed); err != nil {
			return nil, err
		}
	}
	reg.Fields = fields
	// Поле-источник подразделов могли удалить: в БД ссылку обнулил каскад
	// (ON DELETE SET NULL), в снимке для события обнуляем её сами.
	if reg.SectionFieldID != nil && findField(fields, *reg.SectionFieldID) == nil {
		reg.SectionFieldID = nil
	}
	s.publish(ctx, id, "registry:updated", registryPayload(reg))
	return reg, nil
}

// stripRemovedFields — удалить значения отключённых полей из всех записей и
// пересчитать search_text по актуальному набору полей.
func (s *Service) stripRemovedFields(ctx context.Context, reg *domain.Registry, fields []domain.Field, removed []int64) error {
	recs, err := s.repo.AllRecords(ctx, reg.ID)
	if err != nil {
		return err
	}
	var orphans []string
	for _, rec := range recs {
		changed := false
		for _, fid := range removed {
			key := domain.FieldID(fid)
			if v, ok := rec.Data[key]; ok {
				orphans = append(orphans, filePaths(v)...)
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
		userID, companyID := quotaScope(reg)
		s.files.RemoveFor(ctx, userID, companyID, orphans)
	}
	return nil
}

func registryPayload(r *domain.Registry) map[string]any {
	return map[string]any{
		"id": r.ID, "owner_id": r.OwnerID, "company_id": r.CompanyID,
		"team_access": r.TeamAccess, "name": r.Name,
		"position": r.Position, "section_field_id": r.SectionFieldID,
		"accounting": r.Accounting, "fields": r.Fields,
	}
}
