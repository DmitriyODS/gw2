// Package dto — формы REST-ответов petsvc.
package dto

import (
	"math"
	"sort"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ISO-форматы как у marshmallow: datetime с офсетом, date — YYYY-MM-DD.
func isoTime(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000000+00:00") }
func isoDate(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format("2006-01-02")
	return &s
}

func isoTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := isoTime(*t)
	return &s
}

type QuestDTO struct {
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Hint     string `json:"hint"`
	Unit     string `json:"unit"`
	Target   int    `json:"target"`
	Progress int    `json:"progress"`
	Done     bool   `json:"done"`
	Claimed  bool   `json:"claimed"`
	Reward   int    `json:"reward"`
}

// PetDTO — снапшот питомца. Контекстные поля (feeds_left, phrase, evolved…)
// добавляются по месту использования.
type PetDTO struct {
	UserID           int64           `json:"user_id"`
	Name             string          `json:"name"`
	Species          string          `json:"species"`
	Stage            int             `json:"stage"`
	XP               int             `json:"xp"`
	Kudos            int             `json:"kudos"`
	Hat              *string         `json:"hat"`
	Accessories      []string        `json:"accessories"`
	FeedStreak       int             `json:"feed_streak"`
	LastFedDate      *string         `json:"last_fed_date"`
	User             *domain.UserRef `json:"user,omitempty"`
	NextStageXP      *int            `json:"next_stage_xp"`
	Sick             bool            `json:"sick"`
	// OnVacation — хозяин (и питомец вместе с ним) в отпуске: показатели
	// заморожены, действия и поглаживания недоступны.
	OnVacation bool `json:"on_vacation"`
	Recovery         int             `json:"recovery"`
	RecoveryTarget   int             `json:"recovery_target"`
	// Болезнь: вид, подпись и рецепт (nil — питомец здоров).
	Ailment      *string `json:"ailment"`
	AilmentTitle *string `json:"ailment_title,omitempty"`
	AilmentHint  *string `json:"ailment_hint,omitempty"`
	// RunawayInDays — сколько дней болезни осталось до побега; заполняется
	// только когда пора бить тревогу (RunawayWarnDays).
	RunawayInDays *int `json:"runaway_in_days,omitempty"`
	// Потребности и настроение (среднее шкал; множит XP за работу).
	Needs      domain.NeedValues `json:"needs"`
	Mood       int               `json:"mood"`
	MoodTitle  string            `json:"mood_title"`
	MoodFactor float64           `json:"mood_factor"`
	Personality      *string         `json:"personality"`
	PersonalityTitle *string         `json:"personality_title"`
	UnlockedSpecies  []string        `json:"unlocked_species"`
	Quest            *QuestDTO       `json:"quest"`
	AdventureUntil   *string         `json:"adventure_until"`
	AdventurePlace   *string         `json:"adventure_place"`
	Generation       int                `json:"generation"`
	HouseOwned       []string           `json:"house_owned"`
	HousePlaced      []domain.HouseItem `json:"house_placed"`
	HouseTheme       string             `json:"house_theme"`
	HousePetX        *float64           `json:"house_pet_x,omitempty"`
	HousePetY        *float64           `json:"house_pet_y,omitempty"`

	// Контекстные поля.
	FeedsLeft  *int    `json:"feeds_left,omitempty"`
	FeedsMax   *int    `json:"feeds_max,omitempty"`
	SleepsLeft *int    `json:"sleeps_left,omitempty"`
	SleepsMax  *int    `json:"sleeps_max,omitempty"`
	BathsLeft  *int    `json:"baths_left,omitempty"`
	BathsMax   *int    `json:"baths_max,omitempty"`
	Phrase     *string `json:"phrase,omitempty"`
	Evolved    *bool   `json:"evolved,omitempty"`
	Recovered  *bool   `json:"recovered,omitempty"`
	// Runaway — разовое: питомец сбежал именно на этом запросе (как
	// AdventureReward — фиксируется ровно один раз).
	Runaway *RunawayDTO `json:"runaway,omitempty"`
	// AdventureReward — разовая награда за вернувшееся приключение
	// (только в ответе GetMyPet, зафиксировавшем возврат — как Recovered).
	AdventureReward *AdventureRewardDTO `json:"adventure_reward,omitempty"`
	// StrokesToday — сколько раз ЗРИТЕЛЬ сегодня погладил этого питомца
	// (только в выдаче зоопарка; лимит — domain.StrokeDailyMaxPerPet).
	StrokesToday *int `json:"strokes_today,omitempty"`
}

// AdventureRewardDTO — что принёс питомец из приключения.
type AdventureRewardDTO struct {
	Kudos int    `json:"kudos"`
	XP    int    `json:"xp"`
	Place string `json:"place"`
}

// RunawayDTO — питомец сбежал: кто ушёл, от какой болезни и после скольких
// дней (клиент показывает прощание и новое яйцо).
type RunawayDTO struct {
	Name    string `json:"name"`
	Ailment string `json:"ailment"`
	Days    int    `json:"days"`
}

// NewPet — снапшот питомца для REST-ответов.
func NewPet(p *domain.Pet) *PetDTO {
	dto := &PetDTO{
		UserID:         p.UserID,
		Name:           p.Name,
		Species:        p.Species,
		Stage:          p.Stage,
		XP:             p.XP,
		Kudos:          p.Kudos,
		Hat:            p.Hat,
		Accessories:    orEmpty(p.Accessories),
		FeedStreak:     p.FeedStreak,
		LastFedDate:    isoDate(p.LastFedDate),
		User:           p.User,
		Sick:           p.Sick(),
		OnVacation:     p.OwnerOnVacation,
		Recovery:       p.Recovery,
		RecoveryTarget: domain.RecoveryTarget,
		Needs:          p.Needs,
		Personality:    p.Personality,
		AdventureUntil: isoTimePtr(p.AdventureUntil),
		AdventurePlace: p.AdventurePlace,
		Generation:     max(1, p.Generation),
		HouseOwned:     orEmpty(p.HouseOwned),
		HousePlaced:    orEmptyItems(p.HousePlaced),
		HouseTheme:     p.HouseTheme,
		HousePetX:      p.HousePetX,
		HousePetY:      p.HousePetY,
	}
	if p.Stage < domain.MaxStage {
		next := domain.StageXP[p.Stage+1]
		dto.NextStageXP = &next
	}
	dto.Mood = p.Needs.Mood()
	dto.MoodTitle = domain.MoodTitle(dto.Mood)
	dto.MoodFactor = domain.MoodFactor(dto.Mood)
	if key := p.AilmentKey(); key != "" {
		dto.Ailment = &key
		if a, ok := domain.Ailments[key]; ok {
			title, hint := a.Title, a.Hint
			dto.AilmentTitle, dto.AilmentHint = &title, &hint
		}
		// Предупреждение о побеге — только когда до него рукой подать.
		// Считаем ОСТАВШИЕСЯ дни вверх (питомец, которому осталось 3 дня и
		// 20 часов, ждёт ещё 4 дня, а не 3).
		if p.SickSince != nil {
			until := p.SickSince.AddDate(0, 0, domain.RunawaySickDays)
			left := int(math.Ceil(time.Until(until).Hours() / 24))
			if left <= domain.RunawayWarnDays {
				warn := max(0, left)
				dto.RunawayInDays = &warn
			}
		}
	}
	if p.Personality != nil {
		if pers, ok := domain.Personalities[*p.Personality]; ok {
			title := pers.Title
			dto.PersonalityTitle = &title
		}
	}
	// Доступные облики: всё разблокированное + текущий вид (старые питомцы
	// до миграции могли не иметь его в unlocked).
	unlocked := append([]string{}, p.UnlockedSpecies...)
	if p.Species != "" && p.Species != "egg" && !contains(unlocked, p.Species) {
		unlocked = append(unlocked, p.Species)
	}
	dto.UnlockedSpecies = orEmpty(unlocked)
	dto.Quest = questSnapshot(p)
	return dto
}

func questSnapshot(p *domain.Pet) *QuestDTO {
	if p.QuestKind == nil || p.QuestTarget == nil || *p.QuestTarget == 0 {
		return nil
	}
	var tpl *domain.QuestTemplate
	for i := range domain.QuestTemplates {
		if domain.QuestTemplates[i].Kind == *p.QuestKind {
			tpl = &domain.QuestTemplates[i]
			break
		}
	}
	q := &QuestDTO{
		Kind:    *p.QuestKind,
		Title:   "Дневной квест",
		Target:  *p.QuestTarget,
		Claimed: p.QuestClaimed,
		Reward:  domain.QuestRewardKudos,
	}
	if tpl != nil {
		q.Title, q.Hint, q.Unit = tpl.Title, tpl.Hint, tpl.Unit
	}
	q.Progress = min(p.QuestProgress, q.Target)
	q.Done = q.Progress >= q.Target
	return q
}

type LiveItemDTO struct {
	UnitID    int64           `json:"unit_id"`
	UnitName  string          `json:"unit_name"`
	TaskID    int64           `json:"task_id"`
	TaskName  *string         `json:"task_name"`
	StartedAt string          `json:"started_at"`
	User      *domain.UserRef `json:"user"`
}

type LiveDTO struct {
	Items []*LiveItemDTO `json:"items"`
}

func NewLiveItem(u *domain.ActiveUnit) *LiveItemDTO {
	return &LiveItemDTO{
		UnitID:    u.ID,
		UnitName:  u.Name,
		TaskID:    u.TaskID,
		TaskName:  u.TaskName,
		StartedAt: isoTime(u.StartedAt),
		User:      u.User,
	}
}

// ShopItemDTO — витрина магазина: цена, редкость, окно ротации, остаток
// лимитированного тиража и владение (уже куплено/разблокировано).
type ShopItemDTO struct {
	Key            string  `json:"key"`
	Kind           string  `json:"kind"`
	Rarity         string  `json:"rarity"`
	PriceKudos     int     `json:"price_kudos"`
	UnlockKind     string  `json:"unlock_kind"`
	AchievementKey *string `json:"achievement_key,omitempty"`
	LimitedQuota   *int    `json:"limited_quota,omitempty"`
	Remaining      *int    `json:"remaining,omitempty"`
	SoldOut        bool    `json:"sold_out"`
	ActiveFrom     *string `json:"active_from,omitempty"`
	ActiveTo       *string `json:"active_to,omitempty"`
	Owned          bool    `json:"owned"`
	// Скидка дня: фактическая цена и процент (нет скидки — поля отсутствуют).
	SalePriceKudos *int `json:"sale_price_kudos,omitempty"`
	DiscountPct    *int `json:"discount_pct,omitempty"`
}

// ShopDTO — ответ GET /shop: витрина + признак «сюрприз дня уже получен»
// (иначе фронт после перезагрузки не знает состояние мистери-слота).
type ShopDTO struct {
	Items        []*ShopItemDTO `json:"items"`
	MysteryTaken bool           `json:"mystery_taken"`
}

// ── Сезонный трек ───────────────────────────────────────────────────

// SeasonRewardDTO — порог трека с состоянием для владельца.
type SeasonRewardDTO struct {
	Threshold int    `json:"threshold"`
	Kind      string `json:"kind"` // accessory | decor | kudos
	Key       string `json:"key,omitempty"`
	Amount    int    `json:"amount,omitempty"`
	Reached   bool   `json:"reached"`
	Claimed   bool   `json:"claimed"`
}

// SeasonDTO — состояние сезонного трека: заработано за квартал + пороги.
type SeasonDTO struct {
	Season  string             `json:"season"`  // «2026-Q3»
	EndsAt  string             `json:"ends_at"` // конец квартала (МСК)
	Kudos   int                `json:"kudos"`
	Rewards []*SeasonRewardDTO `json:"rewards"`
}

// NewSeason — снапшот трека: пороги по возрастанию, отметки достигнут/забран.
func NewSeason(season string, endsAt time.Time, earned int, claimedList []int) *SeasonDTO {
	claimed := map[int]bool{}
	for _, t := range claimedList {
		claimed[t] = true
	}
	rewards := make([]*SeasonRewardDTO, 0, len(domain.SeasonTrack))
	for _, r := range domain.SeasonTrack {
		rewards = append(rewards, &SeasonRewardDTO{
			Threshold: r.Threshold,
			Kind:      r.Kind,
			Key:       r.Key,
			Amount:    r.Amount,
			Reached:   earned >= r.Threshold,
			Claimed:   claimed[r.Threshold],
		})
	}
	sort.Slice(rewards, func(i, j int) bool { return rewards[i].Threshold < rewards[j].Threshold })
	return &SeasonDTO{
		Season:  season,
		EndsAt:  isoTime(endsAt),
		Kudos:   earned,
		Rewards: rewards,
	}
}

// ── Домик ───────────────────────────────────────────────────────────

// HouseDecorDTO — позиция каталога декора; price 0 — награда сезонного
// трека (не продаётся).
type HouseDecorDTO struct {
	Key    string `json:"key"`
	Price  int    `json:"price"`
	Owned  bool   `json:"owned"`
	Placed bool   `json:"placed"`
}

// HouseDTO — домик питомца: каталог с владением + текущая расстановка.
type HouseDTO struct {
	Catalog   []*HouseDecorDTO   `json:"catalog"`
	Placed    []domain.HouseItem `json:"placed"`
	PlacedMax int                `json:"placed_max"`
	Theme     string             `json:"theme"`
	PetX      *float64           `json:"pet_x,omitempty"`
	PetY      *float64           `json:"pet_y,omitempty"`
	Kudos     int                `json:"kudos"`
}

// NewHouse — снапшот домика; каталог отсортирован по цене (стабильный
// порядок витрины), сезонные награды — в конце.
func NewHouse(p *domain.Pet) *HouseDTO {
	placedKeys := make([]string, 0, len(p.HousePlaced))
	for _, item := range p.HousePlaced {
		placedKeys = append(placedKeys, item.Key)
	}
	catalog := make([]*HouseDecorDTO, 0, len(domain.HouseDecor))
	for key, price := range domain.HouseDecor {
		catalog = append(catalog, &HouseDecorDTO{
			Key:    key,
			Price:  price,
			Owned:  contains(p.HouseOwned, key),
			Placed: contains(placedKeys, key),
		})
	}
	sort.Slice(catalog, func(i, j int) bool {
		pi, pj := catalog[i].Price, catalog[j].Price
		if (pi == 0) != (pj == 0) {
			return pj == 0 // продаваемые раньше сезонных
		}
		if pi != pj {
			return pi < pj
		}
		return catalog[i].Key < catalog[j].Key
	})
	return &HouseDTO{
		Catalog:   catalog,
		Placed:    orEmptyItems(p.HousePlaced),
		PlacedMax: domain.HousePlacedMax,
		Theme:     p.HouseTheme,
		PetX:      p.HousePetX,
		PetY:      p.HousePetY,
		Kudos:     p.Kudos,
	}
}

// ── Кудо-банк ───────────────────────────────────────────────────────

// BankTierDTO — уровень клиента банка и его условия.
type BankTierDTO struct {
	Key              string `json:"key"`
	Title            string `json:"title"`
	Threshold        int    `json:"threshold"`
	SavingsRatePct   int    `json:"savings_rate_pct"`
	LoanFeePct       int    `json:"loan_fee_pct"`
	LoanMax          int    `json:"loan_max"`
	TransferDailyCap int    `json:"transfer_daily_cap"`
	TransferMax      int    `json:"transfer_max"`
}

func newBankTier(t domain.BankTier) BankTierDTO {
	return BankTierDTO{
		Key: t.Key, Title: t.Title, Threshold: t.Threshold,
		SavingsRatePct: t.SavingsRatePct, LoanFeePct: t.LoanFeePct,
		LoanMax: t.LoanMax, TransferDailyCap: t.TransferDailyCap, TransferMax: t.TransferMax,
	}
}

// GenerousDTO — строка топа щедрости (подарено кудосов за 30 дней).
type GenerousDTO struct {
	User *domain.UserRef `json:"user"`
	Sent int             `json:"sent"`
}

// GoalDTO — копилка-цель.
type GoalDTO struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Emoji    string `json:"emoji"`
	Target   int    `json:"target"`
	Saved    int    `json:"saved"`
	Achieved bool   `json:"achieved"`
}

func NewGoal(g *domain.BankGoal) *GoalDTO {
	return &GoalDTO{ID: g.ID, Title: g.Title, Emoji: g.Emoji,
		Target: g.Target, Saved: g.Saved, Achieved: g.AchievedAt != nil}
}

func NewGoals(goals []*domain.BankGoal) []*GoalDTO {
	out := make([]*GoalDTO, 0, len(goals))
	for _, g := range goals {
		out = append(out, NewGoal(g))
	}
	return out
}

// FundDTO — благотворительный сбор компании.
type FundDTO struct {
	ID          int64           `json:"id"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Emoji       string          `json:"emoji"`
	Target      int             `json:"target"`
	Collected   int             `json:"collected"`
	Status      string          `json:"status"`
	Creator     *domain.UserRef `json:"creator,omitempty"`
	DonorsCount int             `json:"donors_count"`
	MyDonated   int             `json:"my_donated"`
	TopDonors   []*GenerousDTO  `json:"top_donors,omitempty"`
	CreatedAt   string          `json:"created_at"`
	FinishedAt  *string         `json:"finished_at,omitempty"`
}

func NewFund(f *domain.BankFund) *FundDTO {
	d := &FundDTO{
		ID: f.ID, Title: f.Title, Description: f.Description, Emoji: f.Emoji,
		Target: f.Target, Collected: f.Collected, Status: f.Status,
		Creator: f.Creator, DonorsCount: f.DonorsCount, MyDonated: f.MyDonated,
		CreatedAt: isoTime(f.CreatedAt),
	}
	if f.FinishedAt != nil {
		t := isoTime(*f.FinishedAt)
		d.FinishedAt = &t
	}
	for _, g := range f.TopDonors {
		d.TopDonors = append(d.TopDonors, &GenerousDTO{User: g.User, Sent: g.Sent})
	}
	return d
}

func NewFunds(funds []*domain.BankFund) []*FundDTO {
	out := make([]*FundDTO, 0, len(funds))
	for _, f := range funds {
		out = append(out, NewFund(f))
	}
	return out
}

// BankDayStatDTO — приход/расход одного дня (динамика).
type BankDayStatDTO struct {
	Day string `json:"day"` // YYYY-MM-DD (МСК)
	In  int    `json:"in"`
	Out int    `json:"out"`
}

// BankKindStatDTO — приход/расход по виду операции за окно статистики.
type BankKindStatDTO struct {
	Kind string `json:"kind"`
	In   int    `json:"in"`
	Out  int    `json:"out"`
}

// BankStatsDTO — статистика банка за последние BankStatsDays дней.
type BankStatsDTO struct {
	Days     int                `json:"days"`
	Daily    []*BankDayStatDTO  `json:"daily"`
	ByKind   []*BankKindStatDTO `json:"by_kind"`
}

func NewBankStats(daily []domain.BankDayStat, kinds []domain.BankKindStat) *BankStatsDTO {
	d := &BankStatsDTO{Days: domain.BankStatsDays,
		Daily: make([]*BankDayStatDTO, 0, len(daily)), ByKind: make([]*BankKindStatDTO, 0, len(kinds))}
	for _, s := range daily {
		d.Daily = append(d.Daily, &BankDayStatDTO{Day: s.Day.Format("2006-01-02"), In: s.In, Out: s.Out})
	}
	for _, k := range kinds {
		d.ByKind = append(d.ByKind, &BankKindStatDTO{Kind: k.Kind, In: k.In, Out: k.Out})
	}
	return d
}

// BankDTO — сводка кудо-банка владельца.
type BankDTO struct {
	Kudos             int           `json:"kudos"`
	Savings           int           `json:"savings"`
	Loan              int           `json:"loan"`
	Earned            int           `json:"earned"` // заработано за всё время — прогресс уровня
	Tier              BankTierDTO   `json:"tier"`
	NextTier          *BankTierDTO  `json:"next_tier,omitempty"`
	SavingsDailyMax   int           `json:"savings_daily_max"`
	TransferLeftToday int           `json:"transfer_left_today"`
	MonthIn           int           `json:"month_in"`
	MonthOut          int           `json:"month_out"`
	TopGenerous       []*GenerousDTO `json:"top_generous"`
	Goals             []*GoalDTO    `json:"goals"`
	GoalsMax          int           `json:"goals_max"`
	Funds             []*FundDTO    `json:"funds"`
	// InterestPaid — разовое: проценты, начисленные при этом обращении.
	InterestPaid *int `json:"interest_paid,omitempty"`
	// GoalAchieved/FundCompleted — разовые: цель закрыта именно этой операцией.
	GoalAchieved  *GoalDTO `json:"goal_achieved,omitempty"`
	FundCompleted *FundDTO `json:"fund_completed,omitempty"`
}

func NewBank(p *domain.Pet, tier domain.BankTier, next *domain.BankTier,
	earned, monthIn, monthOut int, top []domain.GenerousEntry) *BankDTO {

	d := &BankDTO{
		Kudos: p.Kudos, Savings: p.BankSavings, Loan: p.BankLoan,
		Earned: earned, Tier: newBankTier(tier),
		SavingsDailyMax: domain.SavingsDailyMax,
		MonthIn:         monthIn, MonthOut: monthOut,
		TopGenerous: make([]*GenerousDTO, 0, len(top)),
		Goals:       []*GoalDTO{}, GoalsMax: domain.GoalsMax, Funds: []*FundDTO{},
	}
	if next != nil {
		n := newBankTier(*next)
		d.NextTier = &n
	}
	for _, g := range top {
		d.TopGenerous = append(d.TopGenerous, &GenerousDTO{User: g.User, Sent: g.Sent})
	}
	return d
}

// LedgerEntryDTO — операция выписки.
type LedgerEntryDTO struct {
	ID           int64           `json:"id"`
	Delta        int             `json:"delta"`
	Kind         string          `json:"kind"`
	Comment      string          `json:"comment,omitempty"`
	Counterparty *domain.UserRef `json:"counterparty,omitempty"`
	CreatedAt    string          `json:"created_at"`
}

// LedgerDTO — страница выписки (keyset: next_before_id для «Показать ещё»).
type LedgerDTO struct {
	Items        []*LedgerEntryDTO `json:"items"`
	NextBeforeID *int64            `json:"next_before_id,omitempty"`
}

// NewLedger — страница из pageSize записей; запрос делался на pageSize+1,
// лишняя строка означает «есть ещё».
func NewLedger(entries []*domain.LedgerEntry, pageSize int) *LedgerDTO {
	d := &LedgerDTO{Items: make([]*LedgerEntryDTO, 0, min(len(entries), pageSize))}
	for i, e := range entries {
		if i >= pageSize {
			last := d.Items[len(d.Items)-1].ID
			d.NextBeforeID = &last
			break
		}
		d.Items = append(d.Items, &LedgerEntryDTO{
			ID: e.ID, Delta: e.Delta, Kind: e.Kind, Comment: e.Comment,
			Counterparty: e.Counterparty, CreatedAt: isoTime(e.CreatedAt),
		})
	}
	return d
}

// ActivityLogDTO — запись приватной истории активности питомца.
type ActivityLogDTO struct {
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
	CreatedAt string         `json:"created_at"`
}

func NewActivityLog(e *domain.ActivityLogEntry) *ActivityLogDTO {
	payload := e.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return &ActivityLogDTO{Kind: e.Kind, Payload: payload, CreatedAt: isoTime(e.CreatedAt)}
}

func orEmpty(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func orEmptyItems(s []domain.HouseItem) []domain.HouseItem {
	if s == nil {
		return []domain.HouseItem{}
	}
	return s
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
