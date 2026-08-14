package service

import (
	"context"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

/* Согласие с правовыми документами (см. domain/legal.go).

   Тексты документов лежат на фронте — сервер хранит только ФАКТ согласия:
   какую редакцию человек принял, когда, с какого адреса и какого клиента.
   Этого требует 152-ФЗ: доказывать получение согласия обязан оператор. */

var (
	errLegalVersion = domain.NewError("LEGAL_VERSION_MISMATCH",
		"Документы обновились — перезагрузите страницу и прочитайте новую редакцию", 409)
	errLegalIncomplete = domain.NewError("LEGAL_INCOMPLETE",
		"Нужно принять лицензионное соглашение и дать согласие на обработку персональных данных", 400)
)

// LegalState — что принято и требуется ли согласие (плашка на фронте и карточка
// в настройках).
func (s *Service) LegalState(ctx context.Context, userID int64) (*dto.LegalState, error) {
	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errUserNotFound
	}
	state := &dto.LegalState{
		Version:         domain.LegalVersion,
		Documents:       domain.LegalRequiredDocuments,
		Required:        domain.LegalRequiredFor(u),
		AcceptedVersion: u.LegalVersion,
	}
	if u.LegalAcceptedAt != nil {
		ts := dto.JSONTime(*u.LegalAcceptedAt)
		state.AcceptedAt = &ts
	}
	return state, nil
}

// AcceptLegal — принять действующую редакцию. Возвращает СЕССИЮ: access-токен
// несёт клейм legal_required, и без перевыпуска человек остался бы запертым до
// истечения текущего токена. Refresh не трогаем — сессия та же самая.
//
// activeCompanyID — активная компания из токена: перевыпуск не должен ронять
// выбранную компанию.
func (s *Service) AcceptLegal(ctx context.Context, userID int64, activeCompanyID *int64,
	req dto.LegalAcceptRequest, meta domain.Consent) (*dto.Session, error) {

	if req.Version != domain.LegalVersion {
		return nil, errLegalVersion
	}
	given := map[string]bool{}
	for _, d := range req.Documents {
		given[d] = true
	}
	for _, need := range domain.LegalRequiredDocuments {
		if !given[need] {
			return nil, errLegalIncomplete
		}
	}

	u, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errUserNotFound
	}

	now := time.Now().UTC()
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{
		"legal_version":     domain.LegalVersion,
		"legal_accepted_at": now,
	}); err != nil {
		return nil, err
	}
	meta.UserID = userID
	meta.Version = domain.LegalVersion
	meta.Documents = domain.LegalRequiredDocuments
	if err := s.repo.AddConsent(ctx, meta); err != nil {
		// Журнал — доказательство согласия, и терять его нельзя: откатываем
		// отметку в users, чтобы плашка спросила ещё раз.
		_ = s.repo.UpdateFields(ctx, userID, map[string]any{
			"legal_version": u.LegalVersion, "legal_accepted_at": u.LegalAcceptedAt,
		})
		return nil, err
	}
	s.log.Info("legal.accepted", "user_id", userID, "version", domain.LegalVersion, "ip", meta.IP)

	version := domain.LegalVersion
	u.LegalVersion = &version
	u.LegalAcceptedAt = &now
	return s.session(ctx, u, activeCompanyID, false)
}
