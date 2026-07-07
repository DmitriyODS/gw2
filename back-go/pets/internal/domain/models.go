// Package domain — модели и порты питомцев-грувиков: экономика (XP,
// кудосы), эволюция, магазин (постоянные/ротационные/лимитированные/
// достигаемые предметы), прогулка/лечение/поглаживание, рейтинг признания
// и приватная история активности питомца.
//
// Таблицы (pets, pet_strokes, pet_shop_items, pet_shop_purchases,
// pet_activity_log, pet_kudos_weekly) живут в общей PostgreSQL платформы,
// схему ведёт migrate (goose).
package domain

import "time"

// UserRef — мини-профиль владельца питомца (зоопарк, рейтинг).
type UserRef struct {
	ID         int64   `json:"id"`
	FIO        string  `json:"fio"`
	AvatarPath *string `json:"avatar_path"`
}

// User — пользователь в объёме проверок petsvc.
// Идентичность пользователя — из users; активная компания и роль приходят
// из access-токена и проставляются транспортом, не читаются из users.
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	CompanyID     *int64 // активная компания из токена (не из users)
	RoleLevel     int    // уровень роли в активной компании из токена
	CompanyActive bool
}

// Pet — питомец-грувик. Никогда не деградирует и не умирает; болезнь лишь
// замораживает рост (XP и стадия сохраняются).
type Pet struct {
	UserID          int64
	CompanyID       int64
	Name            string
	Species         string
	Stage           int
	XP              int
	Kudos           int
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
	AdventureUntil  *time.Time // питомец в приключении до этого момента
	AdventurePlace  *string    // локация приключения (для фана)
	User            *UserRef
}

// Away — питомец сейчас в приключении (срок ещё не истёк).
func (p *Pet) Away(now time.Time) bool {
	return p.AdventureUntil != nil && now.Before(*p.AdventureUntil)
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

// FinishedUnit — завершённый юнит (паттерны работы: вид, характер).
type FinishedUnit struct {
	Name  string
	Start time.Time
	End   time.Time
}

// ShopItem — товар магазина питомца (pet_shop_items). Постоянный (без окна
// дат) либо ротационный (active_from/active_to); лимитированный —
// LimitedQuota на компанию; достижимый — UnlockKind="achievement" вместо
// покупки за кудосы.
type ShopItem struct {
	ID             int64
	Key            string
	Kind           string // skin | accessory | species
	Rarity         string // common | rare | epic | legendary
	PriceKudos     int
	UnlockKind     string // shop | achievement
	AchievementKey *string
	LimitedQuota   *int
	ActiveFrom     *time.Time
	ActiveTo       *time.Time
}

// Active — товар доступен сейчас (нет окна дат либо now попадает в него).
func (i *ShopItem) Active(now time.Time) bool {
	if i.ActiveFrom == nil && i.ActiveTo == nil {
		return true
	}
	if i.ActiveFrom != nil && now.Before(*i.ActiveFrom) {
		return false
	}
	if i.ActiveTo != nil && now.After(*i.ActiveTo) {
		return false
	}
	return true
}

// ActivityLogEntry — приватная запись истории активности питомца (замена
// публичной ленты): видна только владельцу. Kind: fed | walked | healed |
// evolved | sickness_started | recovered | item_bought | item_equipped |
// stroked_by.
type ActivityLogEntry struct {
	ID        int64
	PetUserID int64
	Kind      string
	Payload   map[string]any
	CreatedAt time.Time
}
