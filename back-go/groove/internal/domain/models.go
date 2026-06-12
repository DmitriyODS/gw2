// Package domain — модели и порты раздела «Мой Groove»: лента активности,
// реакции, комментарии, кудосы, заряды, питомцы-Грувики, зоопарк, магазин,
// недельные рейды и AI-механики Грувика.
//
// Таблицы (feed_events, feed_reactions, feed_comments, pets, pet_strokes,
// groove_raids) живут в общей PostgreSQL платформы, схему ведёт Alembic.
package domain

import "time"

// UserRef — мини-профиль для ленты/зоопарка (как FeedUserRefSchema).
type UserRef struct {
	ID         int64   `json:"id"`
	FIO        string  `json:"fio"`
	AvatarPath *string `json:"avatar_path"`
}

// User — пользователь в объёме проверок groove (кудосы, поглаживания).
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	CompanyID     *int64
	IsHidden      bool
	RoleLevel     int
	CompanyActive bool
}

// FeedEvent — событие ленты. Kind: unit_started | unit_stopped | task_closed
// | streak | pet_evolved | pet_sick | pet_recovered | kudos | ai_digest
// | raid_started | raid_won | wrapped | quest_done.
type FeedEvent struct {
	ID        int64
	CompanyID int64
	UserID    *int64 // NULL — системное событие (AI-дайджест, рейд)
	Kind      string
	Payload   map[string]any
	CreatedAt time.Time
	User      *UserRef
}

// FeedComment — комментарий события (author NULL + IsBot — Грувик).
type FeedComment struct {
	ID        int64
	EventID   int64
	AuthorID  *int64
	IsBot     bool
	ReplyToID *int64
	Text      string
	CreatedAt time.Time
	Author    *UserRef
}

// Pet — Грувик. Никогда не деградирует и не умирает; болезнь лишь
// замораживает рост (XP и стадия сохраняются).
type Pet struct {
	UserID          int64
	CompanyID       int64
	Name            string
	Species         string
	Stage           int
	XP              int
	Beans           int
	Hat             *string
	Accessories     []string
	FeedStreak      int
	LastFedDate     *time.Time // date
	SickSince       *time.Time
	Recovery        int
	Personality     *string
	UnlockedSpecies []string
	QuestDate       *time.Time // date
	QuestKind       *string
	QuestTarget     *int
	QuestProgress   int
	QuestClaimed    bool
	User            *UserRef
}

// Raid — недельный рейд компании (цель ×1.2 от закрытого на прошлой неделе).
type Raid struct {
	ID         int64
	CompanyID  int64
	WeekStart  time.Time // date (понедельник МСК)
	Boss       string
	Target     int
	Reward     string
	DefeatedAt *time.Time
}

// ActiveUnit — активный юнит для блока «Сейчас в эфире».
type ActiveUnit struct {
	ID        int64
	Name      string
	TaskID    int64
	TaskName  *string
	StartedAt time.Time
	User      *UserRef
}

// FinishedUnit — завершённый юнит (паттерны работы: характер, wrapped).
type FinishedUnit struct {
	Name  string
	Start time.Time
	End   time.Time
}

// StaleTask — засидевшаяся задача для утреннего брифинга.
type StaleTask struct {
	ID             int64
	Name           string
	DepartmentName *string
	ReceivedAt     time.Time
}

// PetConversation — pet-чат мессенджера (read-only: домен msgsvc).
type PetConversation struct {
	ID        int64
	OwnerID   int64 // user_a_id
	CompanyID int64
	IsPetChat bool
}

// UserLocation — локация пользователя для погодных механик Грувика.
type UserLocation struct {
	UserID    int64
	CompanyID int64 // подтягивается из users при чтении
	Lat       float64
	Lon       float64
	City      *string
	UpdatedAt time.Time
}

// Weather — текущая погода (Open-Meteo, код WMO).
type Weather struct {
	Code    int
	TempC   float64
	WindKmh float64
	IsDay   bool
}

// GeoPlace — результат поиска города (Open-Meteo geocoding).
type GeoPlace struct {
	Name    string  `json:"name"`
	Region  *string `json:"region"`
	Country *string `json:"country"`
	Lat     float64 `json:"latitude"`
	Lon     float64 `json:"longitude"`
}

// ChatMessage — сообщение pet-чата из msgsvc (контекст AI-ответа).
type ChatMessage struct {
	IsBot bool
	Text  string
}

// ── Статистика для AI-инструментов Грувика и дайджеста ────────────

type CommonMetrics struct {
	Debt      int
	Received  int
	Closed    int
	Remaining int
}

type EmployeeStat struct {
	UserID     int64
	FIO        string
	TasksCount int
	TotalHours float64
}

type DeptStat struct {
	ID         int64
	Name       string
	TasksCount int
}

type UnitTypeStat struct {
	Name       string
	TotalHours float64
	TasksCount int
}

type CalendarDay struct {
	Date       string
	Received   int
	Closed     int
	TotalHours float64
}
