package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

var (
	errShareNotFound = domain.NewError("NOT_FOUND", "Ссылка не найдена или отозвана", 404)
	// Ссылка «только для своих»: пока гость не вошёл, формы он не увидит.
	errShareAuth = domain.NewError("SHARE_AUTH_REQUIRED",
		"Чтобы открыть форму по этой ссылке, войдите в аккаунт", 401)
)

/* Поделиться формой можно двумя путями:

     ссылка       — код-capability в адресе (плюс необязательный вход);
     адресно      — человеку или всей компании, с уровнем доступа.

   Уровень respond и есть НАЗНАЧЕНИЕ: у адресата появляется обязанность
   ответить (и, если автор задал, срок), а у автора — контроль исполнения.
   Раздавать доступ вправе владелец и уровень edit. */

// ── Адресный доступ (люди и компании) ──

func (s *Service) ListUserShares(ctx context.Context, userID, formID int64) ([]*domain.UserShare, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return nil, err
	}
	return s.repo.ListUserShares(ctx, formID)
}

// ShareTarget — кому выдаём доступ, какой и к какому сроку.
type ShareTarget struct {
	UserID    *int64
	CompanyID *int64
	Access    string
	DueAt     *time.Time
}

// ShareWith — выдать (или изменить) адресный доступ. Компания берётся только из
// тех, где состоит выдающий: назначить форму чужой компании нельзя.
func (s *Service) ShareWith(ctx context.Context, userID, formID int64, targets []ShareTarget) ([]*domain.UserShare, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	form, err := s.require(ctx, a, formID, domain.AccessEdit)
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return nil, domain.ErrNoAudience
	}
	own := map[int64]bool{}
	for _, id := range a.Companies {
		own[id] = true
	}

	// Кого известить: назначение — это поручение, и человек должен узнать о нём
	// сразу, а не при следующем заходе в раздел.
	var notify []int64
	for _, t := range targets {
		switch {
		case t.UserID != nil:
			// Владельцу выдавать нечего: у него и так всё.
			if *t.UserID == form.OwnerID {
				return nil, domain.ErrShareSelf
			}
			notify = append(notify, *t.UserID)
		case t.CompanyID != nil:
			if !own[*t.CompanyID] {
				return nil, domain.ErrForbidden
			}
			members, err := s.users.CompanyMembers(ctx, *t.CompanyID)
			if err != nil {
				return nil, err
			}
			notify = append(notify, members...)
		default:
			return nil, domain.ErrNoAudience
		}
		share := &domain.UserShare{
			FormID: formID, UserID: t.UserID, CompanyID: t.CompanyID,
			Access: domain.NormalizeShareAccess(t.Access), DueAt: t.DueAt,
			CreatedBy: &a.UserID,
		}
		if err := s.repo.PutUserShare(ctx, share); err != nil {
			return nil, err
		}
		if share.Access == domain.AccessRespond {
			s.notifyAssigned(ctx, form, notify, t.DueAt)
		}
		notify = notify[:0]
	}
	// Аудитория изменилась — новые адресаты должны увидеть форму сразу.
	s.publish(ctx, formID, "form:shared", map[string]any{"id": formID})
	return s.repo.ListUserShares(ctx, formID)
}

// notifyAssigned — «вам назначили форму»: тост в приложении и офлайн-пуш
// (pushsvc слушает тот же канал). Себе о собственном назначении не сообщаем.
func (s *Service) notifyAssigned(ctx context.Context, form *domain.Form, userIDs []int64, dueAt *time.Time) {
	recipients := make([]int64, 0, len(userIDs))
	for _, id := range userIDs {
		if id != form.OwnerID {
			recipients = append(recipients, id)
		}
	}
	if len(recipients) == 0 {
		return
	}
	payload := map[string]any{
		"form_id": form.ID, "title": form.Title, "user_ids": recipients,
	}
	if dueAt != nil {
		payload["due_at"] = dueAt
	}
	s.bus.Publish(ctx, "form:assigned", rooms(recipients), payload)
}

// Unshare — отозвать адресный доступ.
func (s *Service) Unshare(ctx context.Context, userID, formID int64, targetUser, targetCompany *int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return err
	}
	// Комнаты берём ДО отзыва: тому, у кого доступ забрали, событие тоже нужно —
	// иначе форма останется висеть в его списке до перезагрузки.
	audience := s.audience(ctx, formID)
	if err := s.repo.DeleteUserShare(ctx, formID, targetUser, targetCompany); err != nil {
		return err
	}
	s.bus.Publish(ctx, "form:shared", audience, map[string]any{"id": formID})
	return nil
}

// Directory — кандидаты в адресаты: коллеги из компаний выдающего.
func (s *Service) Directory(ctx context.Context, userID int64, query string, limit int) ([]*domain.User, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.users.SearchDirectory(ctx, a.Companies, strings.TrimSpace(query), limit)
}

// Companies — компании, которым можно назначить форму (те, где состоит автор).
func (s *Service) Companies(ctx context.Context, userID int64) ([]map[string]any, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(a.Companies))
	for _, id := range a.Companies {
		name, err := s.users.CompanyName(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	return out, nil
}

// ── Внешние ссылки ──

func (s *Service) ListShares(ctx context.Context, userID, formID int64) ([]*domain.Share, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return nil, err
	}
	return s.repo.ListShares(ctx, formID)
}

// ShareParams — настройки внешней ссылки.
type ShareParams struct {
	Name        string
	RequireAuth bool
}

func (s *Service) CreateShare(ctx context.Context, userID, formID int64, p ShareParams) (*domain.Share, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return nil, err
	}
	code, err := records.NewShareCode()
	if err != nil {
		return nil, err
	}
	share := &domain.Share{
		FormID: formID, Code: code, Name: strings.TrimSpace(p.Name),
		RequireAuth: p.RequireAuth, CreatedBy: &userID,
	}
	if err := s.repo.CreateShare(ctx, share); err != nil {
		return nil, err
	}
	return share, nil
}

func (s *Service) UpdateShare(ctx context.Context, userID, formID, shareID int64, p ShareParams) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return err
	}
	return s.repo.UpdateShare(ctx, shareID, formID, strings.TrimSpace(p.Name), p.RequireAuth)
}

func (s *Service) RevokeShare(ctx context.Context, userID, formID, shareID int64) error {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return err
	}
	return s.repo.DeleteShare(ctx, shareID, formID)
}

// ShareVisits — журнал переходов по ссылке.
func (s *Service) ShareVisits(ctx context.Context, userID, formID, shareID int64, limit int) ([]*domain.ShareVisit, error) {
	a, err := s.actor(ctx, userID)
	if err != nil {
		return nil, err
	}
	if _, err := s.require(ctx, a, formID, domain.AccessEdit); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	return s.repo.ListVisits(ctx, shareID, limit)
}

// ── Публичный доступ по коду ──

// Visitor — кто открывает ссылку. У гостя нет аккаунта, поэтому в журнале
// остаётся только адрес и браузер.
type Visitor struct {
	UserID    *int64
	IP        string
	UserAgent string
}

// resolveShare — форма и сама ссылка по коду. Черновик по ссылке не открывается
// вовсе: пока форма не запущена, её содержимое — дело автора.
func (s *Service) resolveShare(ctx context.Context, code string, v Visitor) (*domain.Form, *domain.Share, error) {
	share, err := s.repo.GetShareByCode(ctx, code)
	if err != nil {
		return nil, nil, err
	}
	if share == nil {
		return nil, nil, errShareNotFound
	}
	if share.RequireAuth && (v.UserID == nil || *v.UserID == 0) {
		return nil, nil, errShareAuth
	}
	form, err := s.repo.GetForm(ctx, share.FormID)
	if err != nil {
		return nil, nil, err
	}
	if form == nil {
		return nil, nil, errShareNotFound
	}
	if form.Status == domain.StatusDraft {
		return nil, nil, domain.ErrNotOpen
	}
	return form, share, nil
}

// SharedForm — форма для заполнения по внешней ссылке. Здесь же пишется журнал
// посещений: открытие страницы и есть переход по ссылке.
func (s *Service) SharedForm(ctx context.Context, code string, v Visitor) (*FillView, error) {
	form, share, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	view, err := s.fillView(ctx, form, v.UserID)
	if err != nil {
		return nil, err
	}
	// Журнал ведём best-effort: не записавшийся переход не повод не пустить
	// человека к форме.
	if err := s.repo.LogVisit(ctx, &domain.ShareVisit{
		ShareID: share.ID, UserID: v.UserID, IP: v.IP, UserAgent: v.UserAgent,
	}); err != nil {
		s.log.Warn("forms.visit_log_failed", "share_id", share.ID, "error", err)
	}
	return view, nil
}

// SharedSubmit — ответ, отправленный по внешней ссылке.
func (s *Service) SharedSubmit(ctx context.Context, code string, v Visitor, in SubmitInput) (*SubmitResult, error) {
	form, share, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	in.UserID = v.UserID
	in.ShareID = &share.ID
	in.IP, in.UserAgent = v.IP, v.UserAgent
	return s.submitTo(ctx, form, in)
}

// SharedUpload — файл, приложенный к ответу по внешней ссылке. Место тратится
// из квоты владельца формы: своего аккаунта у гостя может не быть.
func (s *Service) SharedUpload(ctx context.Context, code string, v Visitor, fileName, mime string, data []byte) (*domain.UploadedFile, error) {
	form, _, err := s.resolveShare(ctx, code, v)
	if err != nil {
		return nil, err
	}
	if err := s.acceptable(ctx, form); err != nil {
		return nil, err
	}
	if (v.UserID == nil || *v.UserID == 0) && !form.AllowAnonymous {
		return nil, domain.ErrAuthRequired
	}
	return s.saveUpload(ctx, form, fileName, mime, data)
}
