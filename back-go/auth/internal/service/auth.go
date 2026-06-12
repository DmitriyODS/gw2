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

// ensureCompanyActive — пользователь с привязкой к компании входит только
// в активную; Администраторы системы (без company_id) не блокируются.
func ensureCompanyActive(u *domain.User) error {
	if u.CompanyID == nil {
		return nil
	}
	if u.Company == nil || !u.Company.IsActive {
		var name *string
		if u.Company != nil {
			name = &u.Company.Name
		}
		return domain.NewErrorExtra(
			"COMPANY_DISABLED",
			"Ваша компания отключена. Обратитесь к администратору.",
			403,
			map[string]any{"company_name": name},
		)
	}
	return nil
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
	// Пароль верный — проверяем доступ компании. ПОСЛЕ верификации пароля,
	// чтобы по ответу нельзя было узнать компанию чужого логина.
	if err := ensureCompanyActive(user); err != nil {
		return nil, err
	}

	s.log.Info("auth.login", "user_id", user.ID)
	return s.session(user, true)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*dto.Session, error) {
	userID, err := s.tokens.ParseRefresh(refreshToken)
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
	if err := ensureCompanyActive(user); err != nil {
		return nil, err
	}
	return s.session(user, false)
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
	return s.session(user, true)
}
