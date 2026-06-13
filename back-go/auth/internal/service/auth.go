package service

import (
	"context"
	"fmt"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

func errLocked(seconds int) error {
	return domain.NewErrorExtra(
		"TOO_MANY_ATTEMPTS",
		fmt.Sprintf("Слишком много неудачных попыток. Подождите %d с.", seconds),
		429,
		map[string]any{"retry_after_sec": seconds},
	)
}

var errInvalidCredentials = domain.NewError("INVALID_CREDENTIALS", "Неверный логин или пароль", 401)
var errNoCompanyAccess = domain.NewError("NO_COMPANY_ACCESS", "Нет доступа ни к одной компании. Обратитесь к администратору.", 403)

// isSystemAdmin — Администратор системы: кросс-компанийный, без членств,
// ходит по компаниям через ?company_id=. Только у него роль уровня ADMIN.
func isSystemAdmin(u *domain.User) bool {
	return u.IsRootAdmin || u.Role.Level >= domain.LevelAdmin
}

func (s *Service) Login(ctx context.Context, req dto.LoginRequest) (*dto.Session, error) {
	// Активная блокировка — даже не проверяем пароль.
	if locked := s.throttle.LockRemaining(ctx, req.Login); locked > 0 {
		return nil, errLocked(locked)
	}

	fail := func() error {
		if delay := s.throttle.RegisterFailure(ctx, req.Login); delay > 0 {
			return errLocked(delay)
		}
		return errInvalidCredentials
	}

	user, err := s.repo.GetByLogin(ctx, req.Login)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, fail()
	}

	ok, err := s.repo.VerifyPassword(ctx, req.Password, user.HashPassword)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fail()
	}

	s.throttle.RegisterSuccess(ctx, req.Login)
	s.log.Info("auth.login", "user_id", user.ID)

	// Администратор системы — сессия без компании (контекст через ?company_id=).
	if isSystemAdmin(user) {
		return s.session(ctx, user, nil, true)
	}

	memberships, err := s.repo.ListMemberships(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	switch len(memberships) {
	case 0:
		// Доступа ни к одной компании нет (после верификации пароля, чтобы по
		// ответу нельзя было узнать компанию чужого логина).
		return nil, errNoCompanyAccess
	case 1:
		return s.session(ctx, user, &memberships[0].CompanyID, true)
	default:
		// >1 компании — сначала выбор: возвращаем список и короткий select-токен,
		// полноценную сессию не выдаём (см. SelectCompany).
		selectTok, err := s.tokens.SelectToken(user.ID)
		if err != nil {
			return nil, err
		}
		return &dto.Session{
			UserID:                user.ID,
			NeedsCompanySelection: true,
			SelectToken:           selectTok,
			Companies:             dto.NewMemberships(memberships),
		}, nil
	}
}

// SelectCompany — завершить логин выбором компании (этап после login-gate):
// проверить select-токен и членство, выдать полноценную сессию.
func (s *Service) SelectCompany(ctx context.Context, selectToken string, companyID int64) (*dto.Session, error) {
	userID, err := s.tokens.ParseSelect(selectToken)
	if err != nil {
		return nil, domain.NewError("INVALID_TOKEN", "Сессия выбора компании истекла, войдите заново", 401)
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, domain.NewError("NOT_FOUND", "Пользователь не найден", 401)
	}
	return s.session(ctx, user, &companyID, true)
}

// SwitchCompany — сменить активную компанию в существующей сессии: перевыпуск
// access+refresh с клеймами выбранной компании (роль в ней).
func (s *Service) SwitchCompany(ctx context.Context, userID, companyID int64) (*dto.Session, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, domain.NewError("NOT_FOUND", "Пользователь не найден", 401)
	}
	return s.session(ctx, user, &companyID, true)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*dto.Session, error) {
	userID, companyID, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil, domain.NewError("INVALID_TOKEN", "Refresh token недействителен", 401)
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsHidden {
		return nil, domain.NewError("NOT_FOUND", "Пользователь не найден", 401)
	}
	if isSystemAdmin(user) {
		return s.session(ctx, user, nil, false)
	}
	// Совместимость со старыми refresh-токенами без company_id — берём первичную.
	if companyID == nil {
		companyID = user.CompanyID
	}
	if companyID == nil {
		return nil, errNoCompanyAccess
	}
	return s.session(ctx, user, companyID, false)
}

func (s *Service) ChangeDefault(ctx context.Context, req dto.ChangeDefaultRequest) (*dto.Session, error) {
	if req.NewPassword != req.ConfirmPassword {
		return nil, domain.NewError("PASSWORDS_MISMATCH", "Пароли не совпадают", 400)
	}

	user, err := s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.NewError("NOT_FOUND", "Пользователь не найден", 404)
	}
	if !user.IsDefaultPass {
		return nil, domain.NewError("ALREADY_CHANGED", "Пароль уже был изменён", 422)
	}

	existing, err := s.repo.GetByLogin(ctx, req.NewLogin)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != req.UserID {
		return nil, domain.NewError("LOGIN_TAKEN", "Логин уже занят", 409)
	}

	hashed, err := s.repo.HashPassword(ctx, req.NewPassword)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFields(ctx, user.ID, map[string]any{
		"login": req.NewLogin, "hash_password": hashed, "is_default_pass": false,
	}); err != nil {
		return nil, err
	}

	// Перечитываем — клеймы должны отражать актуальное состояние.
	user, err = s.repo.GetByID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	s.log.Info("auth.change_default", "user_id", user.ID)
	if isSystemAdmin(user) {
		return s.session(ctx, user, nil, true)
	}
	return s.session(ctx, user, user.CompanyID, true)
}
