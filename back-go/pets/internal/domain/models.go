// Package domain — модели и порты питомцев-грувиков: экономика (XP,
// кудосы), эволюция, магазин (постоянные/ротационные/лимитированные/
// достигаемые предметы), прогулка/лечение/поглаживание, рейтинг признания
// и приватная история активности питомца.
//
// Таблицы (pets, pet_strokes, pet_shop_items, pet_shop_purchases,
// pet_activity_log, pet_kudos_weekly) живут в общей PostgreSQL платформы,
// схему ведёт migrate (goose).
package domain

import (
	"encoding/json"
	"time"
)

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
	Ailment         *string // вид болезни (Ailments); NULL ⟺ SickSince == nil
	Recovery        int
	// Потребности: шкалы 0..100 и момент, до которого убывание уже применено
	// (ленивый пересчёт — ApplyNeedsDecay).
	Needs   NeedValues
	NeedsAt time.Time
	Personality     *string
	UnlockedSpecies []string
	QuestDate       *time.Time // date
	QuestKind       *string
	QuestTarget     *int
	QuestProgress   int
	QuestClaimed    bool
	AdventureUntil  *time.Time // питомец в приключении до этого момента
	AdventurePlace  *string    // локация приключения (для фана)
	Generation      int         // престиж: растёт при перерождении, не сбрасывается
	HouseOwned      []string    // купленный декор домика
	HousePlaced     []HouseItem // расставленный декор (⊆ owned, лимит HousePlacedMax)
	HouseTheme      string      // ключ градиентной темы комнаты (HouseThemes)
	HousePetX       *float64    // позиция грувика в сцене (%; NULL — по умолчанию)
	HousePetY       *float64
	// Кудо-банк: вклад/долг меняются только узкими атомарными методами
	// BankRepo (как престиж/домик) — в full-row SavePet не входят.
	BankSavings          int
	BankSavingsAccruedAt *time.Time // с какого момента капает процент (NULL — вклад пуст)
	BankLoan             int        // остаток долга (0 — кредита нет)
	// OwnerOnVacation — хозяин в отпуске (users.on_vacation): питомец тоже
	// отдыхает — показатели заморожены (FreezeClocks), действия недоступны.
	OwnerOnVacation bool
	User            *UserRef
}

// HouseItem — предмет, расставленный в домике: свободные координаты в
// процентах сцены (владелец двигает декор как хочет, не по слотам).
type HouseItem struct {
	Key string  `json:"key"`
	X   float64 `json:"x"` // 0..100, % ширины сцены
	Y   float64 `json:"y"` // 0..100, % высоты сцены
}

// UnmarshalJSON принимает и первую форму расстановки — голую строку-ключ
// (без координат): старые данные/клиенты получают дефолтное место в нижнем
// ряду сцены вместо ошибки валидации.
func (h *HouseItem) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var key string
		if err := json.Unmarshal(b, &key); err != nil {
			return err
		}
		*h = HouseItem{Key: key, X: 50, Y: 78}
		return nil
	}
	type alias HouseItem // без методов — иначе рекурсия UnmarshalJSON
	var a alias
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	*h = HouseItem(a)
	return nil
}

// Away — питомец сейчас в приключении (срок ещё не истёк).
func (p *Pet) Away(now time.Time) bool {
	return p.AdventureUntil != nil && now.Before(*p.AdventureUntil)
}

// Sick — питомец болен (любой болезнью).
func (p *Pet) Sick() bool { return p.SickSince != nil }

// AilmentKey — вид болезни; "" — здоров. Питомцы, заболевшие до появления
// видов болезней, считаются хандрящими (единственная прежняя болезнь).
func (p *Pet) AilmentKey() string {
	if p.SickSince == nil {
		return ""
	}
	if p.Ailment == nil || *p.Ailment == "" {
		return AilmentBlues
	}
	return *p.Ailment
}

// Fall — уложить питомца в болезнь (без сохранения): счётчик лечения с нуля.
func (p *Pet) Fall(ailment string, now time.Time) {
	p.SickSince = &now
	p.Ailment = &ailment
	p.Recovery = 0
}

// Cure — выздороветь (без сохранения).
func (p *Pet) Cure() {
	p.SickSince = nil
	p.Ailment = nil
	p.Recovery = 0
}

// ApplyNeedsDecay — ленивое убывание потребностей к моменту now (без
// сохранения): применяет целые тики, прошедшие с NeedsAt, и сдвигает NeedsAt
// ровно на них — дробный хвост доживает до следующего вызова, поэтому частый
// поллинг клиента не «съедает» убывание. Возвращает true, если состояние
// изменилось (нужно сохранить).
func (p *Pet) ApplyNeedsDecay(now time.Time) bool {
	if p.NeedsAt.IsZero() {
		p.NeedsAt = now
		return true
	}
	ticks := int(now.Sub(p.NeedsAt) / NeedTick)
	if ticks <= 0 {
		return false
	}
	p.NeedsAt = p.NeedsAt.Add(time.Duration(ticks) * NeedTick)
	for _, n := range Needs {
		p.Needs.Add(n.Key, -n.DecayPerTick*ticks)
	}
	return true
}

// FreezeClocks — отпуск владельца: вместо убывания сдвигает NeedsAt вперёд
// целыми тиками (та же кадансность записи, что у ApplyNeedsDecay — частый
// поллинг не гоняет UPDATE) и продлевает SickSince на замороженный интервал:
// после отпуска и убывание шкал, и таймер побега продолжаются с того же
// места, где остановились. Возвращает true, если состояние изменилось.
func (p *Pet) FreezeClocks(now time.Time) bool {
	if p.NeedsAt.IsZero() {
		p.NeedsAt = now
		return true
	}
	ticks := int(now.Sub(p.NeedsAt) / NeedTick)
	if ticks <= 0 {
		return false
	}
	shift := time.Duration(ticks) * NeedTick
	p.NeedsAt = p.NeedsAt.Add(shift)
	if p.SickSince != nil {
		moved := p.SickSince.Add(shift)
		p.SickSince = &moved
	}
	return true
}

// ApplyNeedGains — влияние действия на шкалы (без сохранения).
func (p *Pet) ApplyNeedGains(action string) {
	for key, delta := range NeedGains[action] {
		p.Needs.Add(key, delta)
	}
}

// PendingAilment — болезнь, которую заслужила запущенная потребность
// ("" — ни одна). Здоровье проверяется вызывающим: болеть двумя болезнями
// сразу питомец не умеет.
func (p *Pet) PendingAilment() string {
	for _, n := range Needs {
		if n.Ailment != "" && p.Needs.Get(n.Key) <= 0 {
			return n.Ailment
		}
	}
	return ""
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

// LedgerEntry — операция кудо-банка (pet_kudos_ledger): delta >0 приход,
// <0 расход; kind — источник (unit/task_closed/feed/shop/transfer_in/…);
// Counterparty — второй участник перевода.
type LedgerEntry struct {
	ID             int64
	UserID         int64
	CompanyID      int64
	Delta          int
	Kind           string
	CounterpartyID *int64
	Counterparty   *UserRef
	Comment        string
	CreatedAt      time.Time
}

// GenerousEntry — строка топа щедрости: сколько кудосов человек подарил
// коллегам за период.
type GenerousEntry struct {
	User *UserRef
	Sent int
}

// BankGoal — копилка-цель: личный суб-счёт под конкретную мечту. Кудосы
// лежат в saved (кошелёк уменьшен), процента нет; achieved_at ставится
// однажды при достижении target и назад не снимается.
type BankGoal struct {
	ID         int64
	UserID     int64
	CompanyID  int64
	Title      string
	Emoji      string
	Target     int
	Saved      int
	CreatedAt  time.Time
	AchievedAt *time.Time
}

// BankFund — благотворительный сбор компании: общая цель, куда скидываются
// коллеги. Собранное — потрачено (возвратов нет); status: active → done
// по достижении цели, либо closed при досрочном закрытии.
type BankFund struct {
	ID          int64
	CompanyID   int64
	CreatedBy   *int64
	Creator     *UserRef
	Title       string
	Description string
	Emoji       string
	Target      int
	Collected   int
	Status      string
	CreatedAt   time.Time
	FinishedAt  *time.Time

	DonorsCount int // агрегаты витрины
	MyDonated   int
	TopDonors   []GenerousEntry
}

// BankDayStat — приход/расход за один день (динамика в статистике банка).
type BankDayStat struct {
	Day time.Time
	In  int
	Out int
}

// BankKindStat — суммарный приход/расход по виду операции за окно статистики.
type BankKindStat struct {
	Kind string
	In   int
	Out  int
}
