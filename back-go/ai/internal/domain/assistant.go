package domain

import (
	"context"
	"time"
)

// AssistantConversation — один диалог пользователя с ИИ-ассистентом в
// компании (уникален по паре user_id+company_id — ассистент не ведёт
// историю по чатам, как мессенджер, а держит один непрерывный тред).
type AssistantConversation struct {
	ID        int64
	UserID    int64
	CompanyID int64
	CreatedAt time.Time
}

// AssistantMessage — реплика диалога ассистента (role: user|assistant).
// Sources — провенанс ответа («Данные: …» — какие инструменты реально дали
// факты), только у роли assistant и только если инструменты вызывались.
// MyFeedback — голос владельца диалога (up|down), заполняется в History.
type AssistantMessage struct {
	ID             int64
	ConversationID int64
	Role           string
	Text           string
	Sources        *string
	MyFeedback     *string
	CreatedAt      time.Time
}

const (
	AssistantRoleUser      = "user"
	AssistantRoleAssistant = "assistant"

	AssistantFeedbackUp   = "up"
	AssistantFeedbackDown = "down"
)

// AssistantRepository — персистентность диалога ассистента.
type AssistantRepository interface {
	// GetOrCreateConversation — единственный диалог пользователя в компании
	// (UNIQUE user_id+company_id).
	GetOrCreateConversation(ctx context.Context, userID, companyID int64) (*AssistantConversation, error)
	// RecentMessages — последние N сообщений диалога в ХРОНОЛОГИЧЕСКОМ
	// порядке (старые → новые) — как контекст для tools-цикла.
	RecentMessages(ctx context.Context, conversationID int64, limit int) ([]AssistantMessage, error)
	// History — постраничная лента для REST (новые → старые, курсор by
	// created_at); before=nil — с самого начала (последних сообщений).
	// Заполняет MyFeedback голосом владельца диалога.
	History(ctx context.Context, conversationID int64, limit int, before *time.Time) ([]AssistantMessage, error)
	// AppendMessage — sources=nil для реплик пользователя и ответов без
	// вызова инструментов.
	AppendMessage(ctx context.Context, conversationID int64, role, text string, sources *string) (*AssistantMessage, error)
	// UpsertFeedback — голос 👍/👎 по ответу ассистента; повторный голос той
	// же пары (message, user) заменяет прежний. false без ошибки — сообщения
	// нет, оно не в диалоге (userID, companyID) или это не ответ ассистента.
	UpsertFeedback(ctx context.Context, messageID, userID, companyID int64, verdict string, reason *string) (bool, error)
}

// ── Инструменты ИИ-ассистента: gRPC-клиент tasksvc ────────────────

// StatsSummary — общие метрики компании за период (get_stats_summary).
type StatsSummary struct {
	NewCount        int
	ClosedCount     int
	InProgressCount int
	DebtCount       int
	TotalHours      float64
	PeriodLabel     string
}

// DepartmentStat — отдел + кол-во поступивших задач за период.
type DepartmentStat struct {
	ID       int64
	Name     string
	NewCount int
}

// EmployeeStat — сотрудник + отработанные часы/кол-во задач за период.
type EmployeeStat struct {
	FIO       string
	TaskCount int
	Hours     float64
}

// UnitTypeStat — тип юнита + часы/кол-во задач за период.
type UnitTypeStat struct {
	Name      string
	Hours     float64
	TaskCount int
}

// CalendarDayStat — один день периода: поступило/закрыто/часы.
type CalendarDayStat struct {
	Date        string
	NewCount    int
	ClosedCount int
	Hours       float64
}

// TaskRef — минимум для ссылки на задачу в ответе ассистента.
type TaskRef struct {
	ID             int64
	Name           string
	Color          string
	ResponsibleFIO string
}

// TasksClient — исходящий gRPC tasksvc: статистика и поиск/ссылки задач для
// инструментов ИИ-ассистента (закрывает архитектурный долг старого
// groove-tools.go — данные идут через честный gRPC, а не прямым SQL).
type TasksClient interface {
	GetStatsSummary(ctx context.Context, companyID int64, period string) (*StatsSummary, error)
	ListDepartments(ctx context.Context, companyID int64, period string) ([]DepartmentStat, error)
	GetTopEmployees(ctx context.Context, companyID int64, period string, limit int) ([]EmployeeStat, error)
	GetStatsByUnitTypes(ctx context.Context, companyID int64, period string) ([]UnitTypeStat, error)
	GetStatsCalendar(ctx context.Context, companyID int64, period string) ([]CalendarDayStat, error)
	SearchTasks(ctx context.Context, companyID int64, query string, limit int) ([]TaskRef, error)
	// GetTaskLink — nil без ошибки, если задачи нет либо она не в этой компании.
	GetTaskLink(ctx context.Context, companyID, taskID int64) (*TaskRef, error)
}
