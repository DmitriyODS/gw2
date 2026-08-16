// Package service — бизнес-логика formsvc: формы и опросы, их структура,
// приём ответов, сводка, шаринг и назначения.
//
// Форма принадлежит ЧЕЛОВЕКУ, а не компании: коллеги и компании получают её
// адресным доступом трёх уровней (см. domain/access.go). Поэтому проверка прав
// здесь — не «состоит ли в компании», а «какой у него уровень», и она стоит на
// входе КАЖДОЙ операции.
//
// Сокет-события клиентам публикуются в Redis gw2:forms:events (доставляет
// gatewaysvc) и адресуются АУДИТОРИИ формы поимённо: общей комнаты у раздела
// нет — форма не является достоянием компании.
package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/chunkupload"
)

type Service struct {
	repo    domain.FormRepository
	users   domain.UserReader
	files   domain.FileStore
	bus     domain.EventBus
	uploads *chunkupload.Manager
	log     *slog.Logger
}

type Deps struct {
	Repo    domain.FormRepository
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

func (s *Service) actor(ctx context.Context, userID int64) (Actor, error) {
	companies, err := s.users.CompaniesOf(ctx, userID)
	if err != nil {
		return Actor{}, err
	}
	return Actor{UserID: userID, Companies: companies}, nil
}

/*
require — форма и проверка уровня доступа.

	Отсутствие доступа маскируем под 404: существование чужой формы — само по
	себе сведения, которых спрашивающему знать неоткуда. А вот НЕХВАТКУ уровня
	тому, кто форму уже видит, называем честно: иначе кнопка «Сохранить»
	отвечала бы «не найдено» на открытой перед глазами форме.
*/
func (s *Service) require(ctx context.Context, a Actor, formID int64, want string) (*domain.Form, error) {
	form, err := s.repo.GetForm(ctx, formID)
	if err != nil {
		return nil, err
	}
	if form == nil {
		return nil, domain.ErrFormNotFound
	}
	access, err := s.repo.AccessOf(ctx, formID, a.UserID, a.Companies)
	if err != nil {
		return nil, err
	}
	if access == domain.AccessNone {
		return nil, domain.ErrFormNotFound
	}
	if !domain.AccessAtLeast(access, want) {
		if want == domain.AccessOwner {
			return nil, domain.ErrOwnerOnly
		}
		return nil, domain.ErrForbidden
	}
	form.MyAccess = access
	return form, nil
}

// quotaScope — чья квота платит за файлы ответов: файл формы, заведённой в
// компании, тратит место создателя компании, личной — самого владельца.
func quotaScope(form *domain.Form) (userID, companyID int64) {
	if form.CompanyID != nil {
		return 0, *form.CompanyID
	}
	return form.OwnerID, 0
}

// publish — событие аудитории формы. Уровень доступа в полезной нагрузке НЕ
// едет: у каждого получателя он свой, и общий снимок его бы переврал.
func (s *Service) publish(ctx context.Context, formID int64, event string, payload any) {
	if rooms := s.audience(ctx, formID); len(rooms) > 0 {
		s.bus.Publish(ctx, event, rooms, payload)
	}
}

// audience — комнаты сокет-событий формы: владелец и все, кому она раздана.
func (s *Service) audience(ctx context.Context, formID int64) []string {
	ids, err := s.repo.Audience(ctx, formID)
	if err != nil {
		s.log.Warn("forms.audience_failed", "form_id", formID, "error", err)
		return nil
	}
	return rooms(ids)
}

// rooms — комнаты WS по списку пользователей.
func rooms(ids []int64) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, fmt.Sprintf("user_%d", id))
	}
	return out
}

// formPayload — снимок формы для сокет-события. Структуру не тащим: она нужна
// открытому редактору, а он перечитывает форму сам.
func formPayload(f *domain.Form) map[string]any {
	return map[string]any{
		"id": f.ID, "owner_id": f.OwnerID, "company_id": f.CompanyID,
		"title": f.Title, "description": f.Description, "status": f.Status,
		"quiz": f.Quiz, "position": f.Position, "responses": f.Responses,
		"updated_at": f.UpdatedAt,
	}
}
