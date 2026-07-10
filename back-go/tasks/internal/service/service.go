// Package service — бизнес-логика tasksvc. Портировано из
// back/app/services/{task,unit,stage,comment,stats}_service.py и
// api/{tasks,units,unit_types,stages,departments,stats}.py без изменения
// правил. Сокет-события клиентам публикуются в Redis gw2:tasks:events
// (generic-мост Flask эмитит их в Socket.IO-комнаты вербатим).
package service

import (
	"context"
	"log/slog"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

const roomAll = "all"

type Service struct {
	tasks     domain.TaskRepository
	tags      domain.TagRepository
	units     domain.UnitRepository
	unitTypes domain.UnitTypeRepository
	depts     domain.DepartmentRepository
	stages    domain.StageRepository
	comments  domain.CommentRepository
	stats     domain.StatsRepository
	users     domain.UserReader
	companies domain.CompanyReader
	pets      domain.PetsHooks
	ai        domain.AIClient
	bus       domain.EventBus
	log       *slog.Logger
	// yg — YouGile-модуль (NewYougile привязывает себя сам); nil в тестах
	// ядра — пуш тогда не дёргается.
	yg *Yougile
}

type Deps struct {
	Tasks     domain.TaskRepository
	Tags      domain.TagRepository
	Units     domain.UnitRepository
	UnitTypes domain.UnitTypeRepository
	Depts     domain.DepartmentRepository
	Stages    domain.StageRepository
	Comments  domain.CommentRepository
	Stats     domain.StatsRepository
	Users     domain.UserReader
	Companies domain.CompanyReader
	Pets      domain.PetsHooks
	AI        domain.AIClient
	Bus       domain.EventBus
	Log       *slog.Logger
}

func New(d Deps) *Service {
	return &Service{
		tasks: d.Tasks, tags: d.Tags, units: d.Units, unitTypes: d.UnitTypes,
		depts: d.Depts, stages: d.Stages, comments: d.Comments, stats: d.Stats,
		users: d.Users, companies: d.Companies, pets: d.Pets, ai: d.AI,
		bus: d.Bus, log: d.Log,
	}
}

var errTaskNotFound = domain.NewError("NOT_FOUND", "Задача не найдена", 404)

// taskInCompany — задача по id В СКОУПЕ активной компании актора. Чужая
// компания или отсутствие активной компании неотличимы от несуществующей
// задачи (404) — существование чужих задач не раскрываем.
func (s *Service) taskInCompany(ctx context.Context, taskID int64, companyID *int64) (*domain.Task, error) {
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil || companyID == nil || task.CompanyID != *companyID {
		return nil, errTaskNotFound
	}
	return task, nil
}

// unitInCompany — юнит по id в скоупе активной компании актора (та же
// семантика 404, что и taskInCompany).
func (s *Service) unitInCompany(ctx context.Context, unitID int64, companyID *int64) (*domain.Unit, error) {
	unit, err := s.units.GetUnit(ctx, unitID)
	if err != nil {
		return nil, err
	}
	if unit == nil || companyID == nil || unit.CompanyID != *companyID {
		return nil, errUnitNotFound
	}
	return unit, nil
}

// yougileEnabled — fail-open: ошибка чтения настроек трактуется как
// «интеграция включена» (как дефолт True во Flask).
func (s *Service) yougileEnabled(ctx context.Context, companyID int64) bool {
	enabled, err := s.companies.YougileEnabled(ctx, companyID)
	if err != nil {
		s.log.Warn("tasks.yougile_flag_failed", "company_id", companyID, "error", err)
		return true
	}
	return enabled
}

// enrichTask — дамп одной задачи с полями _enrich_task (поштучные запросы,
// как в одиночных хендлерах Flask).
func (s *Service) enrichTask(ctx context.Context, t *domain.Task, userID int64) (dto.Task, error) {
	isFav, err := s.tasks.IsFavorite(ctx, t.ID, userID)
	if err != nil {
		return dto.Task{}, err
	}
	hasUnits, err := s.tasks.HasAnyUnits(ctx, t.ID)
	if err != nil {
		return dto.Task{}, err
	}
	activeUsers, err := s.tasks.ActiveUsers(ctx, t.ID)
	if err != nil {
		return dto.Task{}, err
	}
	color, err := s.tasks.UserColor(ctx, t.ID, userID)
	if err != nil {
		return dto.Task{}, err
	}
	tags, err := s.tags.TagsByTasks(ctx, []int64{t.ID})
	if err != nil {
		return dto.Task{}, err
	}
	return dto.NewTask(t, dto.TaskEnrich{
		IsFavorite:     isFav,
		HasUnits:       hasUnits,
		ActiveUsers:    activeUsers,
		Color:          color,
		Tags:           tags[t.ID],
		YougileEnabled: s.yougileEnabled(ctx, t.CompanyID),
	}), nil
}

// broadcastTask — сокет-событие task:created/task:updated: тот же дамп без
// личного цвета.
func (s *Service) broadcastTask(ctx context.Context, event string, task dto.Task) {
	s.bus.Publish(ctx, event, []string{roomAll}, dto.NewTaskBroadcast(task))
}
