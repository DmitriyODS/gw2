package domain

import (
	"context"
	"time"
)

// FeedRepo — события ленты, реакции, комментарии. beforeID == 0 — без курсора.
type FeedRepo interface {
	CreateEvent(ctx context.Context, companyID int64, userID *int64, kind string, payload map[string]any) (*FeedEvent, error)
	GetEvent(ctx context.Context, id int64) (*FeedEvent, error)
	ListEvents(ctx context.Context, companyID, beforeID int64, limit int) ([]*FeedEvent, error)

	// ToggleReaction: true — реакция добавлена, false — снята.
	ToggleReaction(ctx context.Context, eventID, userID int64, emoji string) (bool, error)
	ReactionCounts(ctx context.Context, eventIDs []int64) (map[int64]map[string]int, error)
	MyReactions(ctx context.Context, eventIDs []int64, userID int64) (map[int64][]string, error)
	ReactionCountFor(ctx context.Context, eventID int64, emoji string) (int, error)

	CommentCounts(ctx context.Context, eventIDs []int64) (map[int64]int, error)
	ListComments(ctx context.Context, eventID int64) ([]*FeedComment, error)
	CreateComment(ctx context.Context, eventID int64, authorID *int64, text string, replyToID *int64, isBot bool) (*FeedComment, error)
	GetComment(ctx context.Context, id int64) (*FeedComment, error)
	DeleteComment(ctx context.Context, id int64) error

	// Wrapped «Моя неделя».
	CountUserEvents(ctx context.Context, companyID, userID int64, kind string, since time.Time) (int, error)
	ReactionsReceived(ctx context.Context, userID int64, since time.Time) (int, error)
	KudosReceived(ctx context.Context, companyID, userID int64, since time.Time) (int, error)
	// KudosWeekCounts — полученные кудосы по адресатам с момента since
	// (счётчик признания в рейтинге).
	KudosWeekCounts(ctx context.Context, companyID int64, since time.Time) (map[int64]int, error)

	// «Сейчас в эфире»: активные юниты с видимыми владельцами.
	ListActiveUnits(ctx context.Context, companyID int64) ([]*ActiveUnit, error)
}

// PetRepo — питомцы и рейды.
type PetRepo interface {
	GetPet(ctx context.Context, userID int64) (*Pet, error)
	GetOrCreate(ctx context.Context, userID, companyID int64) (*Pet, error)
	SavePet(ctx context.Context, pet *Pet) error
	ListCompanyPets(ctx context.Context, companyID int64) ([]*Pet, error)

	LastUnitEndByUsers(ctx context.Context, userIDs []int64) (map[int64]time.Time, error)
	// SoulmateForUser: (nil, 0, nil) — напарника нет.
	SoulmateForUser(ctx context.Context, userID int64, since time.Time) (*UserRef, int, error)
	FinishedUnitsForUser(ctx context.Context, userID int64, since time.Time, limit int) ([]FinishedUnit, error)

	GetRaid(ctx context.Context, companyID int64, weekStart time.Time) (*Raid, error)
	CreateRaid(ctx context.Context, companyID int64, weekStart time.Time, boss string, target int, reward string) (*Raid, error)
	SetRaidDefeated(ctx context.Context, raidID int64, at time.Time) error
	// GrantRaidRewards: всем питомцам компании +beans и аксессуар reward.
	GrantRaidRewards(ctx context.Context, companyID int64, beans int, reward string) error
	CountClosedBetween(ctx context.Context, companyID int64, start, end time.Time) (int, error)
}

// UserReader — read-only пользователи платформы (владелец — authsvc).
type UserReader interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	// CompanyActive — активна ли выбранная (активная) компания сессии из
	// токена. nil (Администратор системы) → true.
	CompanyActive(ctx context.Context, companyID *int64) (bool, error)
}

// CompanyReader — read-only компании (активность, ai_enabled, выходные,
// режим «Мой Groove»).
type CompanyReader interface {
	ActiveCompanyIDs(ctx context.Context) ([]int64, error)
	AICompanyIDs(ctx context.Context) ([]int64, error)
	// WeekendDays: дни недели 0=Пн … 6=Вс; мусор/отсутствие → дефолт Сб+Вс.
	WeekendDays(ctx context.Context, companyID int64) ([]int, error)
	// GrooveEnabled: включён ли режим «Мой Groove» (settings.uses_groove);
	// отсутствие/мусор → включён.
	GrooveEnabled(ctx context.Context, companyID int64) (bool, error)
}

// WorkReader — read-only задачи/юниты (брифинг, итоги дня, статистика Грувика).
type WorkReader interface {
	CountUserActive(ctx context.Context, userID, companyID int64) (int, error)
	UserStale(ctx context.Context, userID, companyID int64, threshold time.Time, limit int) ([]*StaleTask, error)
	// ActiveUnitForUser: (0, 0, nil) — активного юнита нет.
	ActiveUnitForUser(ctx context.Context, userID int64) (unitID, companyID int64, err error)
	// DaySummary — активность компании за интервал [start, end): юниты,
	// закрытые задачи, часы и лидер по часам (событие «Итоги дня»).
	DaySummary(ctx context.Context, companyID int64, start, end time.Time) (*DaySummaryStats, error)

	CommonMetrics(ctx context.Context, companyID int64, start, end time.Time) (*CommonMetrics, error)
	TopEmployees(ctx context.Context, companyID int64, start, end time.Time) ([]EmployeeStat, error)
	ByDepartments(ctx context.Context, companyID int64, start, end time.Time) ([]DeptStat, error)
	ByUnitTypes(ctx context.Context, companyID int64, start, end time.Time) ([]UnitTypeStat, error)
	Calendar(ctx context.Context, companyID int64, start, end time.Time) ([]CalendarDay, error)
}

// ConversationReader — read-only pet-чаты (домен msgsvc; сами сообщения
// читаются/пишутся ТОЛЬКО через gRPC msgsvc).
type ConversationReader interface {
	GetConversation(ctx context.Context, id int64) (*PetConversation, error)
	// GetPetConversationByOwner: nil — у пользователя ещё нет pet-чата.
	GetPetConversationByOwner(ctx context.Context, ownerID int64) (*PetConversation, error)
}

// LocationRepo — локации пользователей (таблица user_locations, домен groove).
type LocationRepo interface {
	GetLocation(ctx context.Context, userID int64) (*UserLocation, error)
	SaveLocation(ctx context.Context, loc *UserLocation) error
	DeleteLocation(ctx context.Context, userID int64) error
	// ListLocations — локации видимых пользователей активных компаний
	// (обход погодного цикла).
	ListLocations(ctx context.Context) ([]*UserLocation, error)
}

// WeatherProvider — текущая погода и геокодинг (Open-Meteo). Fail-open:
// провайдер недоступен — Грувик просто молчит о погоде.
type WeatherProvider interface {
	Current(ctx context.Context, lat, lon float64) (*Weather, error)
	SearchCities(ctx context.Context, query string, count int) ([]GeoPlace, error)
}

// Daily — дневные счётчики и кэши в Redis. ВСЁ fail-open: Redis лёг —
// лимиты не применяются, кэши пустые, ничего не падает.
type Daily interface {
	// TakeBudget: сколько из want помещается в дневной кап (атомарный резерв).
	TakeBudget(ctx context.Context, userID int64, source string, want, cap int) int
	Left(ctx context.Context, userID int64, source string, cap int) int

	GetCache(ctx context.Context, key string) string
	SetCache(ctx context.Context, key, value string, ttl time.Duration)
	Exists(ctx context.Context, key string) bool
}

// EventPublisher — Socket.IO-события через Flask-мост (gw2:groove:events).
type EventPublisher interface {
	Publish(ctx context.Context, event string, rooms []string, payload any)
}

// AIClient — LLM-шлюз aisvc (gRPC). Fail-open: ИИ выключен/недоступен —
// Enabled=false, потребители уходят в статические фолбэки.
type AIClient interface {
	Enabled(ctx context.Context, companyID int64) bool
	// Chat — один ход chat completion; tools-цикл — ChatWithTools.
	Chat(ctx context.Context, companyID int64, messages []map[string]any,
		maxTokens int, temperature float64, timeout time.Duration) (string, error)
	ChatWithTools(ctx context.Context, companyID int64, messages []map[string]any,
		toolsJSON string, onTool func(name string, args map[string]any) any,
		maxTokens int, temperature float64, timeout time.Duration, maxIterations int) (string, error)
}

// MessengerClient — pet-чат через gRPC msgsvc.
type MessengerClient interface {
	PostBotMessage(ctx context.Context, conversationID int64, text string) error
	ListRecentMessages(ctx context.Context, conversationID int64, limit int) ([]ChatMessage, error)
}
