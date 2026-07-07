package domain

import (
	"context"
	"time"
)

// PetRepo — питомцы: профиль, эволюция, рейтинг признания, поглаживания.
type PetRepo interface {
	GetPet(ctx context.Context, userID int64) (*Pet, error)
	GetOrCreate(ctx context.Context, userID, companyID int64) (*Pet, error)
	SavePet(ctx context.Context, pet *Pet) error
	// AdjustBalances — атомарный инкремент кудосов/XP одним UPDATE (kudos не
	// уходит ниже нуля); возвращает актуальные значения после применения.
	// Начисления хуков (AwardKudos/AwardXP) обязаны идти через него, а не
	// через SavePet: full-row запись из конкурентных горутин перетирает
	// параллельные инкременты устаревшим снимком (lost-update).
	AdjustBalances(ctx context.Context, userID int64, deltaKudos, deltaXP int) (kudos, xp int, err error)
	// SaveEvolution — сохранить ТОЛЬКО поля эволюции (stage/species/
	// personality/unlocked_species), не трогая балансы и квест.
	SaveEvolution(ctx context.Context, pet *Pet) error
	// StartAdventure — узкий UPDATE полей приключения (не full-row SavePet):
	// проставляет срок/локацию, только если питомец не болен и не в пути
	// (false — гонка/уже в пути, ошибки нет).
	StartAdventure(ctx context.Context, userID int64, until time.Time, place string) (bool, error)
	// FinishAdventure — атомарный возврат из приключения: NULL-ит поля,
	// только если срок истёк (WHERE … adventure_until <= now RETURNING) —
	// true отдаётся ровно один раз, двойной GET не начислит награду дважды.
	FinishAdventure(ctx context.Context, userID int64, now time.Time) (place string, returned bool, err error)
	ListCompanyPets(ctx context.Context, companyID int64) ([]*Pet, error)

	LastUnitEndByUsers(ctx context.Context, userIDs []int64) (map[int64]time.Time, error)
	FinishedUnitsForUser(ctx context.Context, userID int64, since time.Time, limit int) ([]FinishedUnit, error)

	// AddWeeklyKudos/WeeklyKudosCounts — счётчик признания рейтинга: сумма
	// кудосов, начисленных с начала текущей ISO-недели (pet_kudos_weekly;
	// беансы тратятся на прогулки/лечение/поглаживания, но «заработанное за
	// неделю» не уменьшается тратой — простой отдельный счётчик проще, чем
	// вычислять его из общей истории начислений).
	AddWeeklyKudos(ctx context.Context, userID int64, isoYear, isoWeek, amount int) error
	WeeklyKudosCounts(ctx context.Context, companyID int64, isoYear, isoWeek int) (map[int64]int, error)

	// StrokesToday/RecordStroke — дневной лимит поглаживаний чужого питомца
	// (таблица pet_strokes: StrokesToday считает строки за день на пару
	// «гладящий → питомец», лимит StrokeDailyMaxPerPet проверяет сервис).
	StrokesToday(ctx context.Context, petOwnerID, strokerID int64, day time.Time) (int, error)
	// StrokesTodayByStroker — сколько раз гладящий сегодня погладил КАЖДОГО
	// питомца (map[владелец]count) — витрине зоопарка, чтобы «наглажен до
	// завтра» переживал перезагрузку страницы.
	StrokesTodayByStroker(ctx context.Context, strokerID int64, day time.Time) (map[int64]int, error)
	RecordStroke(ctx context.Context, petOwnerID, strokerID int64, day time.Time) error
}

// ShopRepo — магазин питомца: постоянные/ротационные/лимитированные/
// достигаемые предметы (pet_shop_items, pet_shop_purchases).
type ShopRepo interface {
	// ListActiveItems — товары, активные сейчас (без окна дат либо now
	// попадает в active_from/active_to).
	ListActiveItems(ctx context.Context, now time.Time) ([]*ShopItem, error)
	GetItem(ctx context.Context, key string) (*ShopItem, error)
	CountPurchases(ctx context.Context, itemID, companyID int64) (int, error)
	// RecordPurchase — запись покупки; при quota != nil проверка остатка
	// тиража и INSERT выполняются атомарно (одна транзакция под локом
	// товара), исчерпанный тираж → ErrSoldOut. Проверка COUNT на сервисном
	// уровне — только превентивная подсказка витрины, авторитетна эта.
	RecordPurchase(ctx context.Context, itemID, companyID, userID int64, quota *int) error
}

// ActivityRepo — приватная история активности питомца (pet_activity_log).
type ActivityRepo interface {
	Append(ctx context.Context, petUserID int64, kind string, payload map[string]any) error
	ListForPet(ctx context.Context, petUserID int64, limit int) ([]*ActivityLogEntry, error)
}

// UserReader — read-only пользователи платформы (владелец — authsvc).
type UserReader interface {
	GetUser(ctx context.Context, id int64) (*User, error)
	// IsCompanyMember — состоит ли пользователь в компании (скоуп
	// поглаживания чужого питомца рамками компании).
	IsCompanyMember(ctx context.Context, userID, companyID int64) (bool, error)
	// CompanyActive — активна ли выбранная (активная) компания сессии из
	// токена. nil (Администратор системы) → true.
	CompanyActive(ctx context.Context, companyID *int64) (bool, error)
}

// CompanyReader — read-only компании (активность, выходные, режим «Мой Groove»).
type CompanyReader interface {
	ActiveCompanyIDs(ctx context.Context) ([]int64, error)
	// WeekendDays: дни недели 0=Пн … 6=Вс; мусор/отсутствие → дефолт Сб+Вс.
	WeekendDays(ctx context.Context, companyID int64) ([]int, error)
	// GrooveEnabled: включён ли режим «Мой Groove» (settings.uses_groove);
	// отсутствие/мусор → включён.
	GrooveEnabled(ctx context.Context, companyID int64) (bool, error)
}

// WorkReader — read-only юниты для блока «Сейчас в эфире».
type WorkReader interface {
	ListActiveUnits(ctx context.Context, companyID int64) ([]*ActiveUnit, error)
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

// EventPublisher — сокет-события через gatewaysvc (gw2:pets:events).
type EventPublisher interface {
	Publish(ctx context.Context, event string, rooms []string, payload any)
}

// PortalClient — системный пост-поздравление в корпоративном портале
// (gRPC portalsvc CreateSystemPost). Fire-and-forget: реализация сама
// уходит в горутину с таймаутом, ошибки — только в лог (недоступный
// portalsvc гейм-механику не роняет; дедуп повторов — на стороне
// portalsvc). Поле в Service nil-able — без клиента посты не публикуются.
type PortalClient interface {
	CreateSystemPost(companyID, authorUserID int64, systemKind, title, body string)
}
