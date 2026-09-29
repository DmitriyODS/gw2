// Package service — бизнес-логика portalsvc: корпоративный портал компании
// (посты, комментарии, реакции, закрепление, тематические разделы, пересылка
// в мессенджер). Полностью независим от питомцев-грувиков (petsvc). Топики
// ведёт администратор компании, посты/комментарии/реакции — любой участник.
// Сокет-события клиентам публикуются в Redis gw2:portal:events (доставляет
// gatewaysvc).
package service

import (
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/pkg/billingclient"
	"github.com/DmitriyODS/gw2/back-go/pkg/events"
	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

// companyRoom — события портала уходят только участникам компании с ней
// активной (комнату сажает шлюз по токену), а не в общую "all".
func companyRoom(companyID int64) []string { return []string{events.CompanyRoom(companyID)} }

type Service struct {
	repo      domain.Repository
	files     domain.FileStore
	bus       domain.EventBus
	messenger domain.MessengerClient
	log       *slog.Logger
	// billing — лимиты тарифа (WithBilling; nil — ограничений нет).
	billing *billingclient.Client
}

type Deps struct {
	Repo      domain.Repository
	Files     domain.FileStore
	Bus       domain.EventBus
	Messenger domain.MessengerClient
	Log       *slog.Logger
}

func New(d Deps) *Service {
	return &Service{
		repo: d.Repo, files: d.Files,
		bus: d.Bus, messenger: d.Messenger, log: d.Log,
	}
}

// requireTopic — раздел активной компании или доменная 404.
func (s *Service) requireTopic(ctx domain.Ctx, companyID, id int64) (*domain.Topic, error) {
	t, err := s.repo.GetTopic(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil || t.CompanyID != companyID {
		return nil, domain.ErrTopicNotFound
	}
	return t, nil
}

// requirePost — пост активной компании или доменная 404.
func (s *Service) requirePost(ctx domain.Ctx, companyID, id int64) (*domain.Post, error) {
	p, err := s.repo.GetPost(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil || p.CompanyID != companyID {
		return nil, domain.ErrPostNotFound
	}
	return p, nil
}
