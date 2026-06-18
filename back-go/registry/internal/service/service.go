// Package service — бизнес-логика registrysvc: реестры компании, их поля
// (структура карточки) и записи. Структуру правит администратор компании,
// записи — любой её участник. Сокет-события клиентам публикуются в Redis
// gw2:registry:events (доставляет gatewaysvc).
package service

import (
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

const roomAll = "all"

type Service struct {
	repo  domain.RegistryRepository
	files domain.FileStore
	bus   domain.EventBus
	log   *slog.Logger
}

type Deps struct {
	Repo  domain.RegistryRepository
	Files domain.FileStore
	Bus   domain.EventBus
	Log   *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, files: d.Files, bus: d.Bus, log: d.Log}
}

// requireRegistry — реестр активной компании или доменная 404.
func (s *Service) requireRegistry(ctx domain.Ctx, companyID, id int64) (*domain.Registry, error) {
	reg, err := s.repo.GetRegistry(ctx, id)
	if err != nil {
		return nil, err
	}
	if reg == nil || reg.CompanyID != companyID {
		return nil, domain.ErrRegistryNotFound
	}
	return reg, nil
}
