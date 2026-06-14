// Package service — бизнес-логика pushsvc: регистрация токенов устройств и
// рассылка пуш-уведомлений по событиям микросервисов.
package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/push/internal/domain"
)

type Deps struct {
	Tokens   domain.TokenStore
	Users    domain.UserDirectory
	Presence domain.Presence
	Sender   domain.Sender
	Log      *slog.Logger
}

type Service struct {
	tokens   domain.TokenStore
	users    domain.UserDirectory
	presence domain.Presence
	sender   domain.Sender
	log      *slog.Logger
}

func New(d Deps) *Service {
	return &Service{
		tokens: d.Tokens, users: d.Users, presence: d.Presence,
		sender: d.Sender, log: d.Log,
	}
}

func (s *Service) Register(ctx context.Context, userID int64, token, platform string) error {
	if token == "" {
		return errors.New("empty token")
	}
	return s.tokens.Upsert(ctx, domain.DeviceToken{Token: token, UserID: userID, Platform: platform})
}

func (s *Service) Unregister(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.tokens.Delete(ctx, token)
}

// deliver — разослать одно уведомление всем офлайн-получателям из userIDs.
// Заголовок берётся из шаблона n; n.UserID игнорируется. Онлайн-получателей
// пропускаем — их приложение покажет событие вживую (FCM-first).
func (s *Service) deliver(ctx context.Context, userIDs []int64, n domain.Notification) {
	if !s.sender.Enabled() || len(userIDs) == 0 {
		return
	}
	offline, err := s.presence.Offline(ctx, userIDs)
	if err != nil || len(offline) == 0 {
		return
	}
	tokens, err := s.tokens.ListByUsers(ctx, offline)
	if err != nil {
		s.log.Warn("push.list_tokens_failed", "error", err)
		return
	}
	for _, t := range tokens {
		invalid, err := s.sender.Send(ctx, t.Token, n)
		if invalid {
			_ = s.tokens.Delete(ctx, t.Token)
			s.log.Info("push.token_pruned", "user_id", t.UserID)
			continue
		}
		if err != nil {
			s.log.Warn("push.send_failed", "user_id", t.UserID, "error", err)
		}
	}
}
