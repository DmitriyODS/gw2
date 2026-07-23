package service

import (
	"context"
	"strconv"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/dto"
	"github.com/DmitriyODS/gw2/back-go/tasks/internal/report"
)

// stats_activity.go — раздел «Активность»: свою активность видит любой член
// компании («Моя активность»), чужую — только руководитель компании (роль
// Администратор) над членами этой же компании. Всё скоупится активной компанией.

const activityFeedMaxPer = 50

// requireActivityAccess — гард доступа. Своя активность (target == actor) —
// любому члену активной компании; чужая — только администратору компании, и
// цель обязана быть её членом. Возвращает companyID активной компании.
func (s *Service) requireActivityAccess(ctx context.Context, actor *domain.User, targetUserID int64) (int64, error) {
	if actor.CompanyID == nil {
		return 0, domain.NewError("FORBIDDEN", "Нужна активная компания", 403)
	}
	companyID := *actor.CompanyID
	if targetUserID != actor.ID {
		if actor.RoleLevel < domain.LevelAdmin {
			return 0, domain.NewError("FORBIDDEN", "Активность коллег доступна руководителю компании", 403)
		}
		member, err := s.users.IsCompanyMember(ctx, targetUserID, companyID)
		if err != nil {
			return 0, err
		}
		if !member {
			return 0, domain.NewError("NOT_FOUND", "Сотрудник не найден", 404)
		}
	}
	return companyID, nil
}

// EmployeeActivity — сводка активности сотрудника за период.
func (s *Service) EmployeeActivity(ctx context.Context, actor *domain.User, targetUserID int64, start, end time.Time) (*dto.EmployeeActivity, error) {
	companyID, err := s.requireActivityAccess(ctx, actor, targetUserID)
	if err != nil {
		return nil, err
	}
	summary, err := s.stats.EmployeeSummary(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return nil, err
	}
	byTypes, err := s.stats.EmployeeByUnitTypes(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return nil, err
	}
	byWeekday, err := s.stats.EmployeeByWeekday(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return nil, err
	}
	byHour, err := s.stats.EmployeeByHour(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return nil, err
	}
	trend, err := s.stats.EmployeeWeeklyTrend(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return nil, err
	}
	return &dto.EmployeeActivity{
		Period:      period(start, end),
		Summary:     dto.NewActivitySummary(summary),
		ByUnitTypes: dto.NewActivityUnitTypes(byTypes),
		ByWeekday:   dto.NewWeekdayHours(byWeekday),
		ByHour:      dto.NewHourHours(byHour),
		WeeklyTrend: dto.NewWeekPoints(trend),
	}, nil
}

// EmployeeActivityFeed — постраничная лента событий сотрудника (что и когда делал).
func (s *Service) EmployeeActivityFeed(ctx context.Context, actor *domain.User, targetUserID int64, start, end time.Time, page, perPage int) (*dto.ActivityFeed, error) {
	companyID, err := s.requireActivityAccess(ctx, actor, targetUserID)
	if err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > activityFeedMaxPer {
		perPage = activityFeedMaxPer
	}
	items, total, err := s.stats.EmployeeFeed(ctx, companyID, targetUserID, start, end, perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	return &dto.ActivityFeed{
		Items: dto.NewActivityEvents(items), Total: total, Page: page, PerPage: perPage,
	}, nil
}

const activityDocxFeedLimit = 300

var weekdayNames = []string{"Воскресенье", "Понедельник", "Вторник", "Среда", "Четверг", "Пятница", "Суббота"}

var eventLabels = map[string]string{
	"unit_started": "Начал юнит",
	"unit_stopped": "Завершил юнит",
	"task_created": "Создал задачу",
	"task_closed":  "Закрыл задачу",
	"comment":      "Комментарий",
}

// EmployeeActivityDocx — отчёт активности сотрудника в .docx (сводка + разрезы +
// хронология). Возвращает имя сотрудника (для имени файла) и байты документа.
func (s *Service) EmployeeActivityDocx(ctx context.Context, actor *domain.User, targetUserID int64, start, end time.Time) (string, []byte, error) {
	companyID, err := s.requireActivityAccess(ctx, actor, targetUserID)
	if err != nil {
		return "", nil, err
	}
	name := "Сотрудник"
	if u, err := s.users.GetUser(ctx, targetUserID); err == nil && u != nil && u.FIO != "" {
		name = u.FIO
	}
	summary, err := s.stats.EmployeeSummary(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return "", nil, err
	}
	byTypes, err := s.stats.EmployeeByUnitTypes(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return "", nil, err
	}
	weekday, err := s.stats.EmployeeByWeekday(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return "", nil, err
	}
	trend, err := s.stats.EmployeeWeeklyTrend(ctx, companyID, targetUserID, start, end)
	if err != nil {
		return "", nil, err
	}
	feed, _, err := s.stats.EmployeeFeed(ctx, companyID, targetUserID, start, end, activityDocxFeedLimit, 0)
	if err != nil {
		return "", nil, err
	}

	d := report.New()
	d.Heading("Активность сотрудника", 1)
	d.Para(name)
	d.Para("Период: " + start.UTC().Format("02.01.2006") + " — " + end.UTC().Format("02.01.2006"))

	d.Heading("Сводка", 2)
	d.Table([]string{"Показатель", "Значение"}, [][]string{
		{"Отработано часов", num(summary.WorkedHours)},
		{"Создано задач", strconv.Itoa(summary.TasksCreated)},
		{"Закрыто задач", strconv.Itoa(summary.TasksClosed)},
		{"Комментариев", strconv.Itoa(summary.Comments)},
		{"Активных дней", strconv.Itoa(summary.ActiveDays)},
		{"Юнитов", strconv.Itoa(summary.UnitsCount)},
		{"Часов на закрытую задачу", num(summary.AvgHoursPerClosed)},
		{"Среднее время закрытия, ч", num(summary.AvgCycleHours)},
	})

	if len(byTypes) > 0 {
		d.Heading("По типам работ", 2)
		rows := make([][]string, 0, len(byTypes))
		for _, t := range byTypes {
			rows = append(rows, []string{t.Name, num(t.Hours), strconv.Itoa(t.TasksCount)})
		}
		d.Table([]string{"Тип", "Часы", "Задач"}, rows)
	}

	if len(weekday) > 0 {
		d.Heading("По дням недели", 2)
		rows := make([][]string, 0, len(weekday))
		for _, w := range weekday {
			label := "—"
			if w.Weekday >= 0 && w.Weekday < len(weekdayNames) {
				label = weekdayNames[w.Weekday]
			}
			rows = append(rows, []string{label, num(w.Hours)})
		}
		d.Table([]string{"День", "Часы"}, rows)
	}

	if len(trend) > 0 {
		d.Heading("Недельная динамика", 2)
		rows := make([][]string, 0, len(trend))
		for _, p := range trend {
			rows = append(rows, []string{p.Week, num(p.Hours), strconv.Itoa(p.Closed)})
		}
		d.Table([]string{"Неделя", "Часы", "Закрыто"}, rows)
	}

	d.Heading("Хронология", 2)
	if len(feed) == 0 {
		d.Para("Событий за период нет.")
	} else {
		rows := make([][]string, 0, len(feed))
		for _, e := range feed {
			rows = append(rows, []string{
				e.At.UTC().Format("02.01.2006 15:04"),
				eventLabels[e.Type],
				e.TaskName,
				e.Detail,
			})
		}
		d.Table([]string{"Когда", "Действие", "Задача", "Детали"}, rows)
	}

	data, err := d.Bytes()
	if err != nil {
		return "", nil, err
	}
	return name, data, nil
}

func num(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}
