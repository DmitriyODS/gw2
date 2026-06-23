package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

const (
	verificationTTL   = 30 * time.Minute
	resendCooldown    = 60 * time.Second
	maxVerifyAttempts = 5
)

var (
	errInvalidVerification = domain.NewError("INVALID_VERIFICATION", "Неверный или просроченный код подтверждения", 400)
	errVerificationExpired = domain.NewError("VERIFICATION_EXPIRED", "Код подтверждения истёк, запросите новый", 400)
	errTooManyVerify       = domain.NewError("TOO_MANY_ATTEMPTS", "Слишком много попыток, запросите новый код", 429)
)

func errEmailNotVerified(email *string) error {
	extra := map[string]any{}
	if email != nil {
		extra["email"] = *email
	}
	return domain.NewErrorExtra("EMAIL_NOT_VERIFIED", "Подтвердите адрес электронной почты", 403, extra)
}

func errLocked(seconds int) error {
	return domain.NewErrorExtra(
		"TOO_MANY_ATTEMPTS",
		fmt.Sprintf("Слишком много неудачных попыток. Подождите %d с.", seconds),
		429,
		map[string]any{"retry_after_sec": seconds},
	)
}

var errInvalidCredentials = domain.NewError("INVALID_CREDENTIALS", "Неверный логин или пароль", 401)

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
	if user == nil || !user.IsActive {
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
	// Неподтверждённый email — корректные креды, но в систему не пускаем:
	// фронт ведёт на экран подтверждения (с возможностью переотправки).
	if !user.EmailVerified {
		return nil, errEmailNotVerified(user.Email)
	}
	s.log.Info("auth.login", "user_id", user.ID)
	return s.startSession(ctx, user)
}

// Register — публичная регистрация: самостоятельное создание аккаунта без
// компании. Логин генерируется из ФИО (фронт подставляет, пользователь может
// поправить); пустой — генерируем сами. Пароль виден пользователю на фронте
// (без принудительной смены). Сессия НЕ выдаётся — сначала подтверждение email.
func (s *Service) Register(ctx context.Context, req dto.RegisterRequest) (*dto.RegisterResult, error) {
	if err := validateFIO(req.FIO); err != nil {
		return nil, err
	}
	emailPtr, err := normalizeEmail(&req.Email)
	if err != nil {
		return nil, err
	}
	if emailPtr == nil {
		return nil, errValidation("Email обязателен")
	}
	email := *emailPtr
	if err := s.ensureEmailFree(ctx, email, 0); err != nil {
		return nil, err
	}

	login := strings.TrimSpace(req.Login)
	if login == "" {
		if login, err = s.SuggestLogin(ctx, req.FIO); err != nil {
			return nil, err
		}
	} else {
		if err := validateLogin(login); err != nil {
			return nil, err
		}
		if err := s.ensureLoginFree(ctx, login, 0); err != nil {
			return nil, err
		}
	}
	if err := validatePassword(req.Password); err != nil {
		return nil, err
	}

	hashed, err := s.repo.HashPassword(ctx, req.Password)
	if err != nil {
		return nil, err
	}
	user := &domain.User{
		FIO: req.FIO, Login: login, HashPassword: hashed,
		Email: &email, IsDefaultPass: false, EmailVerified: false,
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	// Письмо не ушло — аккаунт создан, пользователь запросит повторную отправку.
	if err := s.sendVerification(ctx, user); err != nil {
		s.log.Warn("auth.verification_send_failed", "user_id", user.ID, "error", err)
	}
	s.log.Info("auth.register", "user_id", user.ID)
	return &dto.RegisterResult{Status: "verification_required", Email: email}, nil
}

// sendVerification — выпуск кода+токена подтверждения и письмо через mailsvc.
func (s *Service) sendVerification(ctx context.Context, user *domain.User) error {
	if user.Email == nil {
		return errValidation("Email обязателен")
	}
	code, err := randomCode()
	if err != nil {
		return err
	}
	tok, err := randomToken()
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.verifications.Upsert(ctx, user.ID, code, tok, now.Add(verificationTTL), now); err != nil {
		return err
	}
	// email в ссылке — чтобы экран подтверждения знал адрес и для ввода кода
	// (без него код-путь падал с «email не задан», работала только ссылка-токен).
	link := strings.TrimRight(s.appBaseURL, "/") + "/verify-email?token=" + tok +
		"&email=" + url.QueryEscape(*user.Email)
	return s.mail.SendVerification(ctx, *user.Email, user.FIO, code, link)
}

func randomCode() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(b[:])%1000000), nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// VerifyEmail — подтверждение по ссылке (token) или вводом кода (email+code).
// Успех помечает email_verified, удаляет запись и выдаёт полноценную сессию.
func (s *Service) VerifyEmail(ctx context.Context, req dto.VerifyEmailRequest) (*dto.Session, error) {
	var v *domain.Verification
	var err error

	if req.Token != "" {
		if v, err = s.verifications.GetByToken(ctx, req.Token); err != nil {
			return nil, err
		}
		if v == nil {
			return nil, errInvalidVerification
		}
	} else {
		if req.Email == "" || req.Code == "" {
			return nil, errInvalidVerification
		}
		u, err := s.repo.GetByEmail(ctx, strings.TrimSpace(req.Email))
		if err != nil {
			return nil, err
		}
		if u == nil {
			return nil, errInvalidVerification
		}
		if v, err = s.verifications.GetByUserID(ctx, u.ID); err != nil {
			return nil, err
		}
		if v == nil {
			return nil, errInvalidVerification
		}
		if v.Attempts >= maxVerifyAttempts {
			return nil, errTooManyVerify
		}
		if v.Code != strings.TrimSpace(req.Code) {
			_ = s.verifications.IncAttempts(ctx, u.ID)
			return nil, errInvalidVerification
		}
	}

	if time.Now().After(v.ExpiresAt) {
		return nil, errVerificationExpired
	}
	user, err := s.repo.GetByID(ctx, v.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errUserNotFound
	}
	if err := s.repo.UpdateFields(ctx, user.ID, map[string]any{"email_verified": true}); err != nil {
		return nil, err
	}
	_ = s.verifications.Delete(ctx, user.ID)
	user.EmailVerified = true
	s.log.Info("auth.email_verified", "user_id", user.ID)
	return s.startSession(ctx, user)
}

// ResendVerification — переотправка письма (троттлинг по last_sent_at).
// Несуществующий/уже подтверждённый email или слишком ранний повтор — тихо ок
// (не раскрываем наличие аккаунта и не спамим почтовый ящик).
func (s *Service) ResendVerification(ctx context.Context, email string) error {
	emailPtr, err := normalizeEmail(&email)
	if err != nil || emailPtr == nil {
		return errValidation("Неверный формат email")
	}
	user, err := s.repo.GetByEmail(ctx, *emailPtr)
	if err != nil {
		return err
	}
	if user == nil || user.EmailVerified {
		return nil
	}
	if v, err := s.verifications.GetByUserID(ctx, user.ID); err == nil && v != nil {
		if time.Since(v.LastSentAt) < resendCooldown {
			return nil
		}
	}
	return s.sendVerification(ctx, user)
}

const passwordResetTTL = time.Hour

var errInvalidReset = domain.NewError("INVALID_RESET", "Ссылка сброса пароля недействительна или истекла", 400)

// RequestPasswordReset — выслать письмо со ссылкой сброса пароля. Несуществующий
// email, неактивный аккаунт или слишком ранний повтор — тихо ок (не раскрываем
// наличие аккаунта). Аккаунт без email сбросить так нельзя (GetByEmail не найдёт).
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	emailPtr, err := normalizeEmail(&email)
	if err != nil || emailPtr == nil {
		return errValidation("Неверный формат email")
	}
	user, err := s.repo.GetByEmail(ctx, *emailPtr)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive || user.Email == nil {
		return nil
	}
	if r, err := s.passwordResets.GetByUserID(ctx, user.ID); err == nil && r != nil {
		if time.Since(r.LastSentAt) < resendCooldown {
			return nil
		}
	}
	tok, err := randomToken()
	if err != nil {
		return err
	}
	now := time.Now()
	if err := s.passwordResets.Upsert(ctx, user.ID, tok, now.Add(passwordResetTTL), now); err != nil {
		return err
	}
	link := strings.TrimRight(s.appBaseURL, "/") + "/reset-password?token=" + tok
	if err := s.mail.SendPasswordReset(ctx, *user.Email, user.FIO, link); err != nil {
		s.log.Warn("auth.reset_send_failed", "user_id", user.ID, "error", err)
	}
	return nil
}

// ResetPassword — установить новый пароль по токену из письма. Возвращает логин
// для префилла на экране входа (после сброса фронт ведёт на login, без автологина).
func (s *Service) ResetPasswordByToken(ctx context.Context, req dto.ResetPasswordRequest) (*dto.PasswordResetResult, error) {
	if req.Token == "" {
		return nil, errInvalidReset
	}
	r, err := s.passwordResets.GetByToken(ctx, req.Token)
	if err != nil {
		return nil, err
	}
	if r == nil || time.Now().After(r.ExpiresAt) {
		return nil, errInvalidReset
	}
	if err := validatePassword(req.NewPassword); err != nil {
		return nil, err
	}
	user, err := s.repo.GetByID(ctx, r.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || !user.IsActive {
		return nil, errInvalidReset
	}
	hashed, err := s.repo.HashPassword(ctx, req.NewPassword)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateFields(ctx, user.ID, map[string]any{
		"hash_password": hashed, "is_default_pass": false,
	}); err != nil {
		return nil, err
	}
	_ = s.passwordResets.Delete(ctx, user.ID)
	s.log.Info("auth.password_reset", "user_id", user.ID)
	return &dto.PasswordResetResult{Login: user.Login}, nil
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
	if user == nil || !user.IsActive {
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
	if user == nil || !user.IsActive {
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
	if user == nil || !user.IsActive {
		return nil, domain.NewError("NOT_FOUND", "Пользователь не найден", 401)
	}
	if user.IsSuperAdmin {
		return s.session(ctx, user, nil, false)
	}
	// Активной компании могло не стать (вышел/исключён) — переходим в сессию без
	// компании, а не роняем refresh.
	if companyID != nil {
		m, err := s.repo.GetMembership(ctx, userID, *companyID)
		if err != nil {
			return nil, err
		}
		if m == nil {
			companyID = nil
		}
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
	return s.startSession(ctx, user)
}
