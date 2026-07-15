package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/auth/internal/dto"
)

// normalizeCode — код вводят руками (ТВ-сценарий): приводим к верхнему регистру
// и убираем пробелы/дефисы, которыми его удобно показывать группами.
func normalizeCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	return strings.NewReplacer(" ", "", "-", "").Replace(code)
}

// linkTTL — окно жизни кода спаривания: коротко, чтобы засвеченный код быстро
// протухал; approve перезаписывает состояние с тем же TTL, давая инициатору
// время забрать сессию.
const linkTTL = 2 * time.Minute

// linkAlphabet — читаемый набор для короткого кода (без похожих 0/O/1/I).
// Длина 32 делит 256 нацело — modulo-смещения при генерации нет.
const linkAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

const linkCodeLen = 6

var (
	errLinkExpired     = domain.NewError("LINK_EXPIRED", "Код устарел, обновите его", 404)
	errLinkUsed        = domain.NewError("LINK_ALREADY_USED", "Этот код уже подтверждён другим аккаунтом", 409)
	errLinkForbidden   = domain.NewError("LINK_FORBIDDEN", "Недопустимый запрос", 403)
	errLinkNeedCompany = domain.NewError("LINK_NEED_COMPANY", "Сначала выберите компанию, под которой авторизовать ТВ-киоск", 409)
)

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func genLinkCode() (string, error) {
	b := make([]byte, linkCodeLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, linkCodeLen)
	for i := range b {
		out[i] = linkAlphabet[int(b[i])%len(linkAlphabet)]
	}
	return string(out), nil
}

// freshLinkCode — свободный код (короткий → на всякий случай проверяем занятость).
func (s *Service) freshLinkCode(ctx context.Context) (string, error) {
	for tries := 0; tries < 5; tries++ {
		code, err := genLinkCode()
		if err != nil {
			return "", err
		}
		existing, err := s.link.Get(ctx, code)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return code, nil
		}
	}
	return "", domain.NewError("LINK_BUSY", "Не удалось сгенерировать код, попробуйте ещё раз", 503)
}

// LinkStart — инициатор (устройство без входа / ТВ-киоск) заводит спаривание:
// возвращает публичный code (показать/в QR) и приватный secret (держит только
// инициатор, им же забирает сессию в LinkClaim). Секрет в БД не хранится.
func (s *Service) LinkStart(ctx context.Context, kind string) (*dto.LinkStartResult, error) {
	if kind != domain.LinkKindTV {
		kind = domain.LinkKindLogin
	}
	code, err := s.freshLinkCode(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := randomHex(24)
	if err != nil {
		return nil, err
	}
	dl := domain.DeviceLink{Kind: kind, SecretHash: hashSecret(secret), Status: domain.LinkStatusPending}
	if err := s.link.Save(ctx, code, dl, linkTTL); err != nil {
		return nil, err
	}
	return &dto.LinkStartResult{Code: code, Secret: secret, Kind: kind, ExpiresInSec: int(linkTTL.Seconds())}, nil
}

// LinkInfo — тип и статус кода (для экрана подтверждения; без секрета).
func (s *Service) LinkInfo(ctx context.Context, code string) (*dto.LinkInfo, error) {
	dl, err := s.link.Get(ctx, normalizeCode(code))
	if err != nil {
		return nil, err
	}
	if dl == nil {
		return nil, errLinkExpired
	}
	return &dto.LinkInfo{Kind: dl.Kind, Status: dl.Status}, nil
}

// LinkApprove — авторизованный пользователь подтверждает спаривание. Для tv
// киоск получит активную компанию подтверждающего (её обязательно выбрать).
// Идемпотентно для того же пользователя; чужое повторное подтверждение — отказ.
func (s *Service) LinkApprove(ctx context.Context, code string, userID int64, activeCompanyID *int64) error {
	code = normalizeCode(code)
	dl, err := s.link.Get(ctx, code)
	if err != nil {
		return err
	}
	if dl == nil {
		return errLinkExpired
	}
	if dl.Status == domain.LinkStatusApproved {
		if dl.UserID == userID {
			return nil
		}
		return errLinkUsed
	}
	if dl.Kind == domain.LinkKindTV && activeCompanyID == nil {
		return errLinkNeedCompany
	}
	dl.Status = domain.LinkStatusApproved
	dl.UserID = userID
	if dl.Kind == domain.LinkKindTV {
		dl.CompanyID = activeCompanyID
	}
	return s.link.Save(ctx, code, *dl, linkTTL)
}

// LinkClaim — инициатор опрашивает статус; после approve отдаёт сессию
// (одноразово — код гасится до выпуска токенов). Секрет обязателен: только
// инициатор знает его, поэтому засветивший код/QR посторонний сессию не заберёт.
// tv-спаривание входит сразу в выбранную компанию, login — обычным путём
// (с login-gate при нескольких компаниях).
func (s *Service) LinkClaim(ctx context.Context, code, secret string) (*dto.LinkClaimResult, error) {
	dl, err := s.link.Get(ctx, normalizeCode(code))
	if err != nil {
		return nil, err
	}
	if dl == nil {
		return &dto.LinkClaimResult{Status: "expired"}, nil
	}
	if dl.SecretHash != hashSecret(secret) {
		return nil, errLinkForbidden
	}
	if dl.Status != domain.LinkStatusApproved {
		return &dto.LinkClaimResult{Status: "pending"}, nil
	}
	if err := s.link.Delete(ctx, normalizeCode(code)); err != nil {
		return nil, err
	}
	u, err := s.repo.GetByID(ctx, dl.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		return &dto.LinkClaimResult{Status: "expired"}, nil
	}
	var sess *dto.Session
	if dl.CompanyID != nil {
		sess, err = s.session(ctx, u, dl.CompanyID, true)
	} else {
		sess, err = s.startSession(ctx, u)
	}
	if err != nil {
		return nil, err
	}
	return &dto.LinkClaimResult{Status: "ok", Session: sess}, nil
}
