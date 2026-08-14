// Package service — бизнес-логика registrysvc: реестры, их поля (структура
// карточки), записи, учёт выдач и шаринг.
//
// Реестр принадлежит ЧЕЛОВЕКУ, а не компании: коллеги и компании получают его
// адресным доступом трёх уровней (см. domain/access.go). Поэтому проверка прав
// здесь — не «состоит ли в компании», а «какой у него уровень», и она стоит на
// входе КАЖДОЙ операции.
//
// Сокет-события клиентам публикуются в Redis gw2:registry:events (доставляет
// gatewaysvc) и адресуются АУДИТОРИИ реестра поимённо: общей комнаты у раздела
// больше нет — реестр перестал быть общим достоянием компании.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
	"github.com/DmitriyODS/gw2/back-go/registry/internal/domain"
)

type Service struct {
	repo    domain.RegistryRepository
	users   domain.UserReader
	files   domain.FileStore
	bus     domain.EventBus
	uploads *chunkupload.Manager
	log     *slog.Logger
	// billing — лимиты тарифа (WithBilling; nil — ограничений нет).
	billing *billingclient.Client
}

type Deps struct {
	Repo    domain.RegistryRepository
	Users   domain.UserReader
	Files   domain.FileStore
	Bus     domain.EventBus
	Uploads *chunkupload.Manager
	Log     *slog.Logger
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, users: d.Users, files: d.Files, bus: d.Bus,
		uploads: d.Uploads, log: d.Log}
}

// Actor — кто выполняет операцию. Компании нужны для доступа через шары: их
// список считается один раз на запрос, а не на каждую проверку.
type Actor struct {
	UserID    int64
	Companies []int64
}

// actor — собрать контекст выполняющего.
func (s *Service) actor(ctx context.Context, userID int64) (Actor, error) {
	companies, err := s.users.CompaniesOf(ctx, userID)
	if err != nil {
		return Actor{}, err
	}
	return Actor{UserID: userID, Companies: companies}, nil
}

/*
require — реестр и проверка уровня доступа.

	Отсутствие доступа маскируем под 404: существование чужого реестра — само по
	себе сведения, которых спрашивающему знать неоткуда. А вот НЕХВАТКУ уровня
	тому, кто реестр уже видит, называем честно: иначе кнопка «Сохранить»
	отвечала бы «не найдено» на существующей перед глазами записи.
*/
func (s *Service) require(ctx context.Context, a Actor, registryID int64, want string) (*domain.Registry, error) {
	reg, err := s.repo.GetRegistry(ctx, registryID)
	if err != nil {
		return nil, err
	}
	if reg == nil {
		return nil, domain.ErrRegistryNotFound
	}
	access, err := s.repo.AccessOf(ctx, registryID, a.UserID, a.Companies)
	if err != nil {
		return nil, err
	}
	if access == domain.AccessNone {
		return nil, domain.ErrRegistryNotFound
	}
	if !domain.AccessAtLeast(access, want) {
		if want == domain.AccessOwner {
			return nil, domain.ErrOwnerOnly
		}
		return nil, domain.ErrForbidden
	}
	reg.MyAccess = access
	return reg, nil
}

// quotaScope — чья квота платит за файлы реестра: файл компанийного реестра
// тратит место создателя компании, личного — самого владельца.
func quotaScope(reg *domain.Registry) (userID, companyID int64) {
	if reg.CompanyID != nil {
		return 0, *reg.CompanyID
	}
	return reg.OwnerID, 0
}

// publish — событие аудитории реестра. Уровень доступа в полезной нагрузке НЕ
// едет: у каждого получателя он свой, и общий снимок его бы переврал.
func (s *Service) publish(ctx context.Context, registryID int64, event string, payload any) {
	if rooms := s.audience(ctx, registryID); len(rooms) > 0 {
		s.bus.Publish(ctx, event, rooms, payload)
	}
}

// audience — комнаты сокет-событий реестра: владелец и все, кому он раздан.
func (s *Service) audience(ctx context.Context, registryID int64) []string {
	ids, err := s.repo.Audience(ctx, registryID)
	if err != nil {
		s.log.Warn("registry.audience_failed", "registry_id", registryID, "error", err)
		return nil
	}
	rooms := make([]string, 0, len(ids))
	for _, id := range ids {
		rooms = append(rooms, fmt.Sprintf("user_%d", id))
	}
	return rooms
}
