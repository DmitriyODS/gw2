package service

import (
	"context"
	"strings"
	"unicode"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

/* Экран блокировки.

   Отошёл от компьютера — приложение закрывается пин-кодом, но СЕССИЯ остаётся
   живой: выход из аккаунта потерял бы открытые окна, черновики и позицию в
   разделах, ради чего блокировкой никто не пользовался бы.

   Пин проверяет СЕРВЕР и хранит его хешем (pgcrypto, как пароли): проверка в
   браузере снималась бы правкой localStorage, а короткий код в открытом виде
   опасен при утечке таблицы. Пароль от аккаунта тоже подходит — иначе
   забытый пин означал бы выход из системы на всех устройствах. */

const (
	lockPinMinLen = 4
	lockPinMaxLen = 8
	// lockMaxIdleMin — потолок задержки: сутки бездействия это уже не «отошёл».
	lockMaxIdleMin = 24 * 60
)

var errBadPin = domain.NewError("BAD_PIN", "Пин-код — от 4 до 8 цифр", 400)

// ScreenLockState — состояние блокировки для настроек.
func (s *Service) ScreenLockState(ctx context.Context, userID int64) (*dto.ScreenLock, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}
	return &dto.ScreenLock{
		Enabled:  user.LockPinHash != nil && *user.LockPinHash != "",
		AfterMin: user.LockAfterMin,
	}, nil
}

// SetScreenLock — включить блокировку с пин-кодом или сменить задержку.
// Пустой пин при включённой блокировке меняет только задержку.
func (s *Service) SetScreenLock(ctx context.Context, userID int64, req dto.ScreenLockRequest) (*dto.ScreenLock, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}

	updates := map[string]any{}
	if pin := strings.TrimSpace(req.Pin); pin != "" {
		if !validPin(pin) {
			return nil, errBadPin
		}
		hashed, err := s.repo.HashPassword(ctx, pin)
		if err != nil {
			return nil, err
		}
		updates["lock_pin_hash"] = hashed
	} else if user.LockPinHash == nil {
		// Включить блокировку без пина нельзя: снимать её было бы нечем.
		return nil, errBadPin
	}

	if req.AfterMin != nil {
		minutes := *req.AfterMin
		if minutes < 0 || minutes > lockMaxIdleMin {
			return nil, domain.NewError("VALIDATION",
				"Задержка блокировки — от 1 минуты до суток", 400)
		}
		if minutes == 0 {
			updates["lock_after_min"] = nil // только вручную
		} else {
			updates["lock_after_min"] = minutes
		}
	}

	if err := s.repo.UpdateFields(ctx, userID, updates); err != nil {
		return nil, err
	}
	s.log.Info("lock.enabled", "user_id", userID)
	return s.ScreenLockState(ctx, userID)
}

// DisableScreenLock — снять блокировку целиком (нужен действующий пин или
// пароль: иначе её отключил бы любой, кто подошёл к запертому экрану).
func (s *Service) DisableScreenLock(ctx context.Context, userID int64, secret string) error {
	if _, err := s.checkLockSecret(ctx, userID, secret); err != nil {
		return err
	}
	if err := s.repo.UpdateFields(ctx, userID, map[string]any{
		"lock_pin_hash": nil, "lock_after_min": nil,
	}); err != nil {
		return err
	}
	s.log.Info("lock.disabled", "user_id", userID)
	return nil
}

// UnlockScreen — снять запертый экран пином или паролем аккаунта.
func (s *Service) UnlockScreen(ctx context.Context, userID int64, secret string) error {
	_, err := s.checkLockSecret(ctx, userID, secret)
	return err
}

/* checkLockSecret — сверить пин ИЛИ пароль аккаунта. Пароль принимаем всегда:
   пин забывается чаще, чем пароль, и без запасного пути человеку осталось бы
   только выйти из аккаунта на всех устройствах. */
func (s *Service) checkLockSecret(ctx context.Context, userID int64, secret string) (*domain.User, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errUserNotFound
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errWrongPin
	}
	if user.LockPinHash != nil && *user.LockPinHash != "" {
		if ok, err := s.repo.VerifyPassword(ctx, secret, *user.LockPinHash); err != nil {
			return nil, err
		} else if ok {
			return user, nil
		}
	}
	ok, err := s.repo.VerifyPassword(ctx, secret, user.HashPassword)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errWrongPin
	}
	return user, nil
}

var errWrongPin = domain.NewError("WRONG_PIN", "Неверный пин-код", 403)

func validPin(pin string) bool {
	if len(pin) < lockPinMinLen || len(pin) > lockPinMaxLen {
		return false
	}
	for _, r := range pin {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
