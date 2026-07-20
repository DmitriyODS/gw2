package service

import (
	"context"
	"strings"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
)

// _REINDEX_FIELDS — поля задачи, влияющие на текст эмбеддинга (формирует
// aisvc); изменение любого из них в UpdateTask перегенерит эмбеддинг.
var reindexFields = map[string]bool{
	"name": true, "department_id": true, "responsible_user_id": true,
}

// ensureNotOnVacation — гард режима «в отпуске» (users.on_vacation): актор в
// отпуске не создаёт/не правит задачи и не запускает юниты. Системные пути
// (YouGile-вебхук) идут мимо — они работают репозиторием, не этими методами.
func (s *Service) ensureNotOnVacation(ctx context.Context, actorID int64) error {
	user, err := s.users.GetUser(ctx, actorID)
	if err != nil {
		return err
	}
	if user != nil && user.OnVacation {
		return domain.NewError("ON_VACATION",
			"Вы в отпуске — создание и редактирование задач недоступно", 403)
	}
	return nil
}

func (s *Service) validateResponsible(ctx context.Context, userID *int64, companyID int64) error {
	if userID == nil {
		return nil
	}
	user, err := s.users.GetUser(ctx, *userID)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive {
		return domain.NewError("USER_NOT_FOUND", "Сотрудник не найден", 404)
	}
	// Принадлежность — по членству в АКТИВНОЙ компании (user_companies).
	member, err := s.users.IsCompanyMember(ctx, *userID, companyID)
	if err != nil {
		return err
	}
	if !member {
		return domain.NewError("USER_FOREIGN", "Сотрудник из другой компании", 422)
	}
	return nil
}

func (s *Service) validateStage(ctx context.Context, stageID *int64, companyID int64) error {
	if stageID == nil {
		return nil
	}
	stage, err := s.stages.GetStage(ctx, *stageID)
	if err != nil {
		return err
	}
	if stage == nil {
		return domain.NewError("STAGE_NOT_FOUND", "Этап не найден", 404)
	}
	if stage.CompanyID != companyID {
		return domain.NewError("STAGE_FOREIGN", "Этап принадлежит другой компании", 422)
	}
	return nil
}

// ListTasks — список с фильтрами и батч-обогащением. Поиск: если у компании
// включён AI — целиком семантический по проиндексированным задачам, иначе
// LIKE по названию (никаких гибридов — см. комментарий в api/tasks.py).
func (s *Service) ListTasks(ctx context.Context, f domain.TaskListFilter) (*dto.TaskList, error) {
	search := strings.TrimSpace(f.Search)
	if search != "" && f.CompanyID != nil && s.ai.Enabled(ctx, *f.CompanyID) {
		hits := s.ai.SemanticSearch(ctx, *f.CompanyID, search)
		if hits == nil {
			hits = []int64{}
		}
		// При включённом AI всегда отдаём семантическую выдачу — даже пустую.
		f.OrderedIDs, f.OrderedSet = hits, true
	}

	items, total, err := s.tasks.ListTasks(ctx, f)
	if err != nil {
		return nil, err
	}

	taskIDs := make([]int64, 0, len(items))
	for _, t := range items {
		taskIDs = append(taskIDs, t.ID)
	}
	enrich, err := s.tasks.Enrichment(ctx, taskIDs, f.CurrentUserID)
	if err != nil {
		return nil, err
	}
	tagsByTask, err := s.tags.TagsByTasks(ctx, taskIDs)
	if err != nil {
		return nil, err
	}
	ygEnabled := true
	if f.CompanyID != nil {
		ygEnabled = s.yougileEnabled(ctx, *f.CompanyID)
	}

	out := make([]dto.Task, 0, len(items))
	for _, t := range items {
		var color *string
		if c, ok := enrich.UserColors[t.ID]; ok {
			c := c
			color = &c
		}
		out = append(out, dto.NewTask(t, dto.TaskEnrich{
			IsFavorite:     enrich.FavoriteIDs[t.ID],
			HasUnits:       enrich.WithUnits[t.ID],
			ActiveUsers:    enrich.ActiveUsers[t.ID],
			Color:          color,
			Tags:           tagsByTask[t.ID],
			YougileEnabled: ygEnabled,
		}))
	}
	return &dto.TaskList{Items: out, Page: f.Page, PerPage: f.PerPage, Total: total}, nil
}

func (s *Service) GetTask(ctx context.Context, taskID, userID int64) (*dto.Task, error) {
	task, err := s.tasks.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, errTaskNotFound
	}
	out, err := s.enrichTask(ctx, task, userID)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetTaskInCompany — задача по id для REST: только в активной компании актора.
func (s *Service) GetTaskInCompany(ctx context.Context, taskID, userID int64, companyID *int64) (*dto.Task, error) {
	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}
	out, err := s.enrichTask(ctx, task, userID)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// createTaskCore — создание задачи с бизнес-проверками, без дампа и
// сокет-события (общая часть CreateTask и YouGile-импорта).
func (s *Service) createTaskCore(ctx context.Context, actorID, companyID int64, req dto.TaskCreate) (*domain.Task, error) {
	if err := s.ensureNotOnVacation(ctx, actorID); err != nil {
		return nil, err
	}
	dept, err := s.depts.GetDepartment(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}
	if dept == nil {
		return nil, domain.NewError("DEPT_NOT_FOUND", "Отдел не найден", 404)
	}
	if dept.CompanyID != companyID {
		return nil, domain.NewError("DEPT_FOREIGN", "Отдел принадлежит другой компании", 422)
	}

	// По умолчанию ответственный = автор задачи.
	responsible := req.ResponsibleUserID
	if responsible == nil {
		responsible = &actorID
	}
	if err := s.validateResponsible(ctx, responsible, companyID); err != nil {
		return nil, err
	}
	if err := s.validateStage(ctx, req.StageID, companyID); err != nil {
		return nil, err
	}

	task := &domain.Task{
		Name:              req.Name,
		AuthorID:          actorID,
		DepartmentID:      req.DepartmentID,
		CompanyID:         companyID,
		LinkYougile:       req.LinkYougile,
		Deadline:          req.Deadline,
		ResponsibleUserID: responsible,
		StageID:           req.StageID,
	}
	if req.ReceivedAt != nil {
		task.ReceivedAt = *req.ReceivedAt
	}
	if err := s.tasks.CreateTask(ctx, task); err != nil {
		return nil, err
	}
	s.log.Info("task.create", "task_id", task.ID, "author_id", actorID)
	s.ai.ScheduleReindex(task.ID)
	return task, nil
}

func (s *Service) CreateTask(ctx context.Context, actorID, companyID int64, req dto.TaskCreate) (*dto.Task, error) {
	task, err := s.createTaskCore(ctx, actorID, companyID, req)
	if err != nil {
		return nil, err
	}
	out, err := s.GetTask(ctx, task.ID, actorID)
	if err != nil {
		return nil, err
	}
	s.broadcastTask(ctx, "task:created", *out)
	return out, nil
}

func (s *Service) UpdateTask(ctx context.Context, taskID, actorID int64, companyID *int64, req dto.TaskUpdate) (*dto.Task, error) {
	if err := s.ensureNotOnVacation(ctx, actorID); err != nil {
		return nil, err
	}
	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}

	fields := map[string]any{}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.LinkYougileSet {
		fields["link_yougile"] = req.LinkYougile
	}
	if req.DepartmentID != nil {
		dept, err := s.depts.GetDepartment(ctx, *req.DepartmentID)
		if err != nil {
			return nil, err
		}
		if dept == nil {
			return nil, domain.NewError("DEPT_NOT_FOUND", "Отдел не найден", 404)
		}
		if dept.CompanyID != task.CompanyID {
			return nil, domain.NewError("DEPT_FOREIGN", "Отдел принадлежит другой компании", 422)
		}
		fields["department_id"] = *req.DepartmentID
	}
	if req.ReceivedAtSet {
		fields["received_at"] = req.ReceivedAt
	}
	if req.DeadlineSet {
		fields["deadline"] = req.Deadline
	}
	if req.ResponsibleSet {
		if err := s.validateResponsible(ctx, req.ResponsibleUserID, task.CompanyID); err != nil {
			return nil, err
		}
		fields["responsible_user_id"] = req.ResponsibleUserID
	}
	if req.StageSet {
		if err := s.validateStage(ctx, req.StageID, task.CompanyID); err != nil {
			return nil, err
		}
		fields["stage_id"] = req.StageID
	}
	if err := s.tasks.UpdateTaskFields(ctx, taskID, fields); err != nil {
		return nil, err
	}

	changed := req.ChangedFields()
	for _, f := range changed {
		if reindexFields[f] {
			s.ai.ScheduleReindex(taskID)
			break
		}
	}

	out, err := s.GetTask(ctx, taskID, actorID)
	if err != nil {
		return nil, err
	}
	// Исходящий пуш в YouGile — best-effort в фоне, антицикл через sync_hash.
	if s.yg != nil {
		s.yg.PushAfterUpdate(taskID, actorID, changed)
	}
	s.broadcastTask(ctx, "task:updated", *out)
	return out, nil
}

func (s *Service) DeleteTask(ctx context.Context, taskID int64, companyID *int64) error {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return err
	}
	if err := s.tasks.DeleteTask(ctx, taskID); err != nil {
		return err
	}
	s.log.Info("task.delete", "task_id", taskID)
	s.bus.Publish(ctx, "task:deleted", []string{roomAll}, map[string]any{"task_id": taskID})
	return nil
}

func (s *Service) ArchiveTask(ctx context.Context, taskID, actorID int64, companyID *int64) (*dto.Task, error) {
	if err := s.ensureNotOnVacation(ctx, actorID); err != nil {
		return nil, err
	}
	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}
	if task.IsArchived {
		return nil, domain.NewError("ALREADY_ARCHIVED", "Задача уже архивирована", 422)
	}
	hasActive, err := s.tasks.HasActiveUnit(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if hasActive {
		return nil, domain.NewError("HAS_ACTIVE_UNIT",
			"Нельзя архивировать задачу с активным юнитом", 422)
	}

	now := time.Now().UTC()
	if err := s.tasks.UpdateTaskFields(ctx, taskID,
		map[string]any{"is_archived": true, "archived_at": now}); err != nil {
		return nil, err
	}
	s.log.Info("task.archive", "task_id", taskID)

	task.IsArchived, task.ArchivedAt = true, &now
	s.pets.OnTaskClosed(task, actorID)

	out, err := s.GetTask(ctx, taskID, actorID)
	if err != nil {
		return nil, err
	}
	if s.yg != nil {
		s.yg.PushAfterArchive(taskID, actorID, true)
	}
	s.bus.Publish(ctx, "task:archived", []string{roomAll}, map[string]any{
		"task_id": taskID, "archived_at": dto.ISO(now),
	})
	return out, nil
}

func (s *Service) RestoreTask(ctx context.Context, taskID, actorID int64, companyID *int64) (*dto.Task, error) {
	if err := s.ensureNotOnVacation(ctx, actorID); err != nil {
		return nil, err
	}
	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}
	if !task.IsArchived {
		return nil, domain.NewError("NOT_ARCHIVED", "Задача не архивирована", 422)
	}
	if err := s.tasks.UpdateTaskFields(ctx, taskID,
		map[string]any{"is_archived": false, "archived_at": nil}); err != nil {
		return nil, err
	}
	s.log.Info("task.restore", "task_id", taskID)
	s.ai.ScheduleReindex(taskID)

	out, err := s.GetTask(ctx, taskID, actorID)
	if err != nil {
		return nil, err
	}
	if s.yg != nil {
		s.yg.PushAfterArchive(taskID, actorID, false)
	}
	s.bus.Publish(ctx, "task:restored", []string{roomAll}, map[string]any{"task_id": taskID})
	return out, nil
}

// SetResponsible / SetStage — отдельные PATCH-роуты v3 (двигают задачу и
// шлют общий task:updated).
func (s *Service) SetResponsible(ctx context.Context, taskID, actorID int64, companyID *int64, responsibleUserID *int64) (*dto.Task, error) {
	if err := s.ensureNotOnVacation(ctx, actorID); err != nil {
		return nil, err
	}
	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.validateResponsible(ctx, responsibleUserID, task.CompanyID); err != nil {
		return nil, err
	}
	if err := s.tasks.UpdateTaskFields(ctx, taskID,
		map[string]any{"responsible_user_id": responsibleUserID}); err != nil {
		return nil, err
	}
	s.ai.ScheduleReindex(taskID)

	out, err := s.GetTask(ctx, taskID, actorID)
	if err != nil {
		return nil, err
	}
	s.broadcastTask(ctx, "task:updated", *out)
	return out, nil
}

func (s *Service) SetStage(ctx context.Context, taskID, actorID int64, companyID *int64, stageID *int64) (*dto.Task, error) {
	if err := s.ensureNotOnVacation(ctx, actorID); err != nil {
		return nil, err
	}
	task, err := s.taskInCompany(ctx, taskID, companyID)
	if err != nil {
		return nil, err
	}
	if err := s.validateStage(ctx, stageID, task.CompanyID); err != nil {
		return nil, err
	}
	if err := s.tasks.UpdateTaskFields(ctx, taskID,
		map[string]any{"stage_id": stageID}); err != nil {
		return nil, err
	}

	out, err := s.GetTask(ctx, taskID, actorID)
	if err != nil {
		return nil, err
	}
	s.broadcastTask(ctx, "task:updated", *out)
	return out, nil
}

func (s *Service) SetTaskColor(ctx context.Context, taskID, userID int64, companyID *int64, color *string) error {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return err
	}
	return s.tasks.SetUserColor(ctx, taskID, userID, color)
}

func (s *Service) ToggleFavorite(ctx context.Context, taskID, userID int64, companyID *int64) (bool, error) {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return false, err
	}
	return s.tasks.ToggleFavorite(ctx, taskID, userID)
}

func (s *Service) Contributors(ctx context.Context, taskID int64, companyID *int64) ([]dto.UserRef, error) {
	if _, err := s.taskInCompany(ctx, taskID, companyID); err != nil {
		return nil, err
	}
	users, err := s.tasks.Contributors(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return dto.NewUserRefs(users), nil
}
