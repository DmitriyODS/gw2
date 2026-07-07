// Package service — бизнес-логика portalsvc: корпоративный портал компании
// (посты, комментарии, реакции, закрепление, тематические разделы, пересылка
// в мессенджер). Полностью независим от питомцев-грувиков (petsvc). Топики
// ведёт администратор компании, посты/комментарии/реакции — любой участник.
// Сокет-события клиентам публикуются в Redis gw2:portal:events (доставляет
// gatewaysvc).
package service

import (
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/portal/internal/domain"
)

const roomAll = "all"

type Service struct {
	repo      domain.Repository
	users     domain.UserReader
	files     domain.FileStore
	bus       domain.EventBus
	messenger domain.MessengerClient
	log       *slog.Logger
}

type Deps struct {
	Repo      domain.Repository
	Users     domain.UserReader
	Files     domain.FileStore
	Bus       domain.EventBus
	Messenger domain.MessengerClient
	Log       *slog.Logger
}

func New(d Deps) *Service {
	return &Service{
		repo: d.Repo, users: d.Users, files: d.Files,
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
