package service

import (
	"context"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

// Платформенное управление пользователями — раздел «Пользователи» супер-админа.
// Операции над ЧИСТОЙ идентичностью (таблица users), вне контекста компании.
// Доступ гейтится RequireSuperAdmin на роуте; здесь — защита самого супер-админа
// и self-операций. Должность/роль не трогаем (это атрибуты членства в компании).

// CreatePlatformUser — супер-админ заводит самостоятельный аккаунт (без компании).
// Без явного пароля — дефолтный <login>123 и обязательная смена при входе.
func (s *Service) CreatePlatformUser(ctx context.Context, req dto.CreateUserRequest) (*dto.User, error) {
	if err := validateFIO(req.FIO); err != nil {
		return nil, err
	}
	if err := validateLogin(req.Login); err != nil {
		return nil, err
	}
	phone, err := normalizePhone(req.Phone)
	if err != nil {
		return nil, err
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if req.Password != nil {
		if err := validatePassword(*req.Password); err != nil {
			return nil, err
		}
	}
	if err := s.ensureLoginFree(ctx, req.Login, 0); err != nil {
		return nil, err
	}
	if email != nil {
		if err := s.ensureEmailFree(ctx, *email, 0); err != nil {
			return nil, err
		}
	}

	password := req.Login + "123"
	isDefault := true
	if req.Password != nil {
		password = *req.Password
		isDefault = false
	}
	hashed, err := s.repo.HashPassword(ctx, password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		FIO: req.FIO, Login: req.Login, HashPassword: hashed,
		Phone: phone, Email: email, IsDefaultPass: isDefault, EmailVerified: true,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	s.log.Info("platform.user_create", "user_id", user.ID)
	return s.freshUser(ctx, user.ID)
}

// UpdatePlatformUser — правка идентичности любого пользователя (без компании).
func (s *Service) UpdatePlatformUser(ctx context.Context, userID int64, req dto.UpdateUserRequest) (*dto.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}

	updates := map[string]any{}
	if req.FIO != nil {
		if err := validateFIO(*req.FIO); err != nil {
			return nil, err
		}
		updates["fio"] = *req.FIO
	}
	if req.Login != nil {
		if err := validateLogin(*req.Login); err != nil {
			return nil, err
		}
		if err := s.ensureLoginFree(ctx, *req.Login, userID); err != nil {
			return nil, err
		}
		updates["login"] = *req.Login
	}
	if req.Phone != nil {
		phone, err := normalizePhone(req.Phone)
		if err != nil {
			return nil, err
		}
		updates["phone"] = phone
	}
	if req.Email != nil {
		email, err := normalizeEmail(req.Email)
		if err != nil {
			return nil, err
		}
		if email != nil {
			if err := s.ensureEmailFree(ctx, *email, userID); err != nil {
				return nil, err
			}
		}
		updates["email"] = email
	}
	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, userID, updates); err != nil {
			return nil, err
		}
	}
	return s.freshUser(ctx, userID)
}

// ResetPlatformUserPassword — сброс пароля на дефолтный <login>123 с
// обязательной сменой при входе.
func (s *Service) ResetPlatformUserPassword(ctx context.Context, actor *domain.User, userID int64) error {
	if actor != nil && userID == actor.ID {
		return domain.NewError("SELF_RESET", "Нельзя сбросить собственный пароль", 422)
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errUserNotFound
	}
	if user.IsSuperAdmin {
		return errSuperAdminProtected
	}
	hashed, err := s.repo.HashPassword(ctx, user.Login+"123")
	if err != nil {
		return err
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{
		"hash_password": hashed, "is_default_pass": true,
	}); err != nil {
		return err
	}
	s.log.Info("platform.user_reset_password", "user_id", userID)
	return nil
}

// DeactivatePlatformUser — деактивация аккаунта (is_active=false). Данные
// сохраняются (история сообщений, юниты и т.п.), вход блокируется.
func (s *Service) DeactivatePlatformUser(ctx context.Context, actor *domain.User, userID int64) error {
	if actor != nil && userID == actor.ID {
		return domain.NewError("SELF_DEACTIVATE", "Нельзя удалить собственный аккаунт", 422)
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errUserNotFound
	}
	if user.IsSuperAdmin {
		return errSuperAdminProtected
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"is_active": false}); err != nil {
		return err
	}
	s.log.Info("platform.user_deactivate", "user_id", userID)
	return nil
}

// ReactivatePlatformUser — вернуть деактивированный аккаунт (is_active=true):
// вход снова доступен, данные не трогались.
func (s *Service) ReactivatePlatformUser(ctx context.Context, userID int64) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errUserNotFound
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{"is_active": true}); err != nil {
		return err
	}
	s.log.Info("platform.user_reactivate", "user_id", userID)
	return nil
}

// PurgePlatformUser — БЕЗВОЗВРАТНОЕ удаление аккаунта со всеми его данными.
// Разрешено только для уже деактивированного пользователя (сначала «удалить»,
// потом — окончательно), чтобы исключить случайное уничтожение живого аккаунта.
func (s *Service) PurgePlatformUser(ctx context.Context, actor *domain.User, userID int64) error {
	if actor != nil && userID == actor.ID {
		return domain.NewError("SELF_PURGE", "Нельзя удалить собственный аккаунт", 422)
	}
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errUserNotFound
	}
	if user.IsSuperAdmin {
		return errSuperAdminProtected
	}
	if user.IsActive {
		return domain.NewError("PURGE_ACTIVE",
			"Сначала деактивируйте аккаунт, затем удаляйте окончательно", 422)
	}
	if err := s.repo.HardDelete(ctx, userID); err != nil {
		return err
	}
	s.log.Info("platform.user_purge", "user_id", userID)
	return nil
}
