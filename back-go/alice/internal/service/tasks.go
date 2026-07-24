// Сценарии задач и юнитов (tasksvc).
package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/DmitriyODS/gw2/back-go/alice/internal/domain"
)

const noCompanyText = "У вашей связки аккаунтов нет активной компании, поэтому задачи недоступны. " +
	"Войдите в Groove Work, выберите компанию и привяжите аккаунт заново."

func (s *Service) requireCompany(sess *session) *domain.WebhookResponse {
	if sess.companyID == 0 {
		return reply(noCompanyText)
	}
	return nil
}

func (s *Service) taskCreate(ctx context.Context, sess *session, name string, deptID int64) *domain.WebhookResponse {
	if r := s.requireCompany(sess); r != nil {
		return r
	}
	if strings.TrimSpace(name) == "" {
		return reply("Скажите название задачи: например, «добавь задачу подготовить отчёт».")
	}
	task, err := s.tasks.CreateTask(ctx, sess.companyID, sess.userID, name, deptID)
	if err != nil {
		if de := domain.AsDomainError(err); de != nil && de.Code == "DEPARTMENT_REQUIRED" {
			depts, derr := s.tasks.ListDepartments(ctx, sess.companyID)
			if derr != nil {
				return s.errReply(derr)
			}
			st := domain.DialogState{Pending: "choose_department", Kind: "task_create", Title: name}
			names := make([]string, 0, len(depts))
			for _, d := range depts {
				st.Options = append(st.Options, domain.Option{ID: d.ID, Name: d.Name})
				names = append(names, d.Name)
			}
			return replyState("В какой отдел создать задачу?"+enumerate(names), st)
		}
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Готово, создала задачу «%s».", task.Name))
}

// findTask — поиск задачи по названию: 0 — реплика «не нашла», 1 — действие,
// несколько — уточняющий выбор.
func (s *Service) findTask(ctx context.Context, sess *session, query, kind string) (*domain.TaskRef, *domain.WebhookResponse) {
	tasks, err := s.tasks.SearchTasks(ctx, sess.companyID, query, 5)
	if err != nil {
		return nil, s.errReply(err)
	}
	switch len(tasks) {
	case 0:
		return nil, reply(fmt.Sprintf("Не нашла задачу по запросу «%s».", query))
	case 1:
		return &tasks[0], nil
	}
	// Точное совпадение названия важнее широкого поиска — пользователь назвал
	// конкретную задачу (переспрос только при 0 или нескольких точных).
	q := normalize(query)
	var exact *domain.TaskRef
	for i := range tasks {
		if normalize(tasks[i].Name) == q {
			if exact != nil {
				exact = nil
				break
			}
			exact = &tasks[i]
		}
	}
	if exact != nil {
		return exact, nil
	}
	st := domain.DialogState{Pending: "choose_task", Kind: kind}
	names := make([]string, 0, len(tasks))
	for _, t := range tasks {
		st.Options = append(st.Options, domain.Option{ID: t.ID, Name: t.Name})
		names = append(names, t.Name)
	}
	return nil, replyState("Нашла несколько задач. Какая из них?"+enumerate(names), st)
}

func (s *Service) taskClose(ctx context.Context, sess *session, query string) *domain.WebhookResponse {
	if r := s.requireCompany(sess); r != nil {
		return r
	}
	task, r := s.findTask(ctx, sess, query, "task_close")
	if r != nil {
		return r
	}
	st := domain.DialogState{Pending: "confirm_close_task", TaskID: task.ID, Title: task.Name}
	return replyState(fmt.Sprintf("Закрыть задачу «%s»? Скажите «да» или «нет».", task.Name), st)
}

func (s *Service) taskCloseByID(ctx context.Context, sess *session, taskID int64) *domain.WebhookResponse {
	name, err := s.tasks.CloseTask(ctx, sess.companyID, sess.userID, taskID)
	if err != nil {
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Готово, закрыла задачу «%s».", name))
}

func (s *Service) taskList(ctx context.Context, sess *session) *domain.WebhookResponse {
	if r := s.requireCompany(sess); r != nil {
		return r
	}
	tasks, total, err := s.tasks.ListOpenTasks(ctx, sess.companyID, sess.userID, 5)
	if err != nil {
		return s.errReply(err)
	}
	if total == 0 {
		return reply("Открытых задач нет. Отличная работа!")
	}
	names := make([]string, 0, len(tasks))
	for _, t := range tasks {
		names = append(names, t.Name)
	}
	head := fmt.Sprintf("Открытых задач: %d.", total)
	if total > len(tasks) {
		head += fmt.Sprintf(" Вот первые %d:", len(tasks))
	}
	return reply(head + enumerate(names))
}

func (s *Service) unitStart(ctx context.Context, sess *session, query string) *domain.WebhookResponse {
	if r := s.requireCompany(sess); r != nil {
		return r
	}
	task, r := s.findTask(ctx, sess, query, "unit_start")
	if r != nil {
		return r
	}
	return s.unitStartOnTask(ctx, sess, task.ID, task.Name, 0)
}

func (s *Service) unitStartOnTask(ctx context.Context, sess *session, taskID int64, taskName string, typeID int64) *domain.WebhookResponse {
	name, err := s.tasks.StartUnit(ctx, sess.companyID, sess.userID, taskID, typeID)
	if err != nil {
		if de := domain.AsDomainError(err); de != nil && de.Code == "UNIT_TYPE_REQUIRED" {
			types, terr := s.tasks.ListUnitTypes(ctx, sess.companyID)
			if terr != nil {
				return s.errReply(terr)
			}
			st := domain.DialogState{Pending: "choose_unit_type", Kind: "unit_start", TaskID: taskID, Title: taskName}
			names := make([]string, 0, len(types))
			for _, t := range types {
				st.Options = append(st.Options, domain.Option{ID: t.ID, Name: t.Name})
				names = append(names, t.Name)
			}
			return replyState("Каким типом юнита работать?"+enumerate(names), st)
		}
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Поехали! Начала юнит по задаче «%s».", name))
}

func (s *Service) unitStop(ctx context.Context, sess *session) *domain.WebhookResponse {
	stopped, err := s.tasks.StopActiveUnit(ctx, sess.userID)
	if err != nil {
		if de := domain.AsDomainError(err); de != nil && de.Code == "NO_ACTIVE_UNIT" {
			return reply("Сейчас нет активного юнита.")
		}
		return s.errReply(err)
	}
	return reply(fmt.Sprintf("Готово, завершила юнит по задаче «%s» — %s.",
		stopped.TaskName, humanMinutes(stopped.Minutes)))
}

func (s *Service) unitStatus(ctx context.Context, sess *session) *domain.WebhookResponse {
	active, err := s.tasks.GetActiveUnit(ctx, sess.userID)
	if err != nil {
		return s.errReply(err)
	}
	if active == nil {
		return reply("Сейчас нет активного юнита.")
	}
	return reply(fmt.Sprintf("В работе задача «%s», юнит идёт уже %s.",
		active.TaskName, humanMinutes(active.Minutes)))
}

func humanMinutes(m int) string {
	if m < 1 {
		return "меньше минуты"
	}
	if m < 60 {
		return fmt.Sprintf("%d мин", m)
	}
	return fmt.Sprintf("%d ч %d мин", m/60, m%60)
}
