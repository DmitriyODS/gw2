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
	// SaveNeeds — узкое сохранение потребностей и состояния болезни (шкалы,
	// needs_at, ailment/sick_since/recovery): ленивый пересчёт убывания идёт
	// на READ-пути, и full-row SavePet затирал бы там конкурентные начисления
	// хуков (та же причина, что у SaveEvolution).
	SaveNeeds(ctx context.Context, pet *Pet) error
	// AdjustNeeds — атомарный сдвиг шкал (кламп 0..NeedMax) одним UPDATE;
	// возвращает шкалы ПОСЛЕ применения (как AdjustBalances): вызывающий
	// кладёт результат в свой снапшот, а не досчитывает дельту сам — иначе
	// конкурентный сдвиг учтётся дважды.
	AdjustNeeds(ctx context.Context, userID int64, deltas map[string]int) (NeedValues, error)
	// StartAdventure — узкий UPDATE полей приключения (не full-row SavePet):
	// проставляет срок/локацию, только если питомец не болен и не в пути
	// (false — гонка/уже в пути, ошибки нет).
	StartAdventure(ctx context.Context, userID int64, until time.Time, place string) (bool, error)
	// FinishAdventure — атомарный возврат из приключения: NULL-ит поля,
	// только если срок истёк (WHERE … adventure_until <= now RETURNING) —
	// true отдаётся ровно один раз, двойной GET не начислит награду дважды.
	FinishAdventure(ctx context.Context, userID int64, now time.Time) (place string, returned bool, err error)
	// RecallAdventure — досрочный платный возврат: атомарно снимает
	// приключение и списывает cost (guard: в пути и хватает кудосов);
	// false — гонка (уже вернулся) либо недостаток средств.
	RecallAdventure(ctx context.Context, userID int64, cost int) (place string, ok bool, err error)
	// RunAway — побег заброшенного питомца: атомарный сброс ПРОГРЕССА
	// (стадия/XP/вид/характер/стрик/потребности) с guard'ом «болеет дольше
	// sickBefore» в WHERE — конкурентные вызовы (ленивый GET и фоновый цикл)
	// зафиксируют побег ровно один раз. Кудосы, гардероб, домик, банк и
	// поколение остаются. false — гейт не сошёлся.
	RunAway(ctx context.Context, userID int64, sickBefore time.Time) (bool, error)
	// DeletePet — удалить питомца и связанные данные (поглаживания, покупки,
	// недельные кудосы; история — каскадом FK) одной транзакцией.
	DeletePet(ctx context.Context, userID int64) error
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

	// Prestige — атомарное перерождение: generation+1, стадия/XP в ноль,
	// вид в яйцо, только если стадия максимальна, питомец здоров и не в
	// пути (узкий UPDATE — конкурентный хук XP не потеряется и не вернёт
	// стадию). false — условия не сошлись (гонка/повторный клик).
	Prestige(ctx context.Context, userID int64, unlockSpecies string) (generation int, ok bool, err error)

	// AddSeasonalKudos/SeasonalKudos — счётчик сезонного трека: кудосы,
	// начисленные за календарный квартал (pet_kudos_seasonal; по образцу
	// недельного — трата баланса сумму не уменьшает).
	AddSeasonalKudos(ctx context.Context, userID int64, season string, amount int) error
	SeasonalKudos(ctx context.Context, userID int64, season string) (int, error)
	// SeasonClaims/ClaimSeasonReward — забранные пороги трека; Claim
	// атомарен (PK season_claims): false — порог уже забран.
	SeasonClaims(ctx context.Context, userID int64, season string) ([]int, error)
	ClaimSeasonReward(ctx context.Context, userID int64, season string, threshold int) (bool, error)

	// AppendAccessory/AppendHouseDecor — атомарное добавление в jsonb-список
	// (награды трека, покупка декора): не перетирают конкурентные изменения
	// и не дублируют уже имеющийся ключ (false — уже был).
	AppendAccessory(ctx context.Context, userID int64, key string) (bool, error)
	AppendHouseDecor(ctx context.Context, userID int64, key string) (bool, error)
	// BuyHouseDecor — атомарная покупка декора: списывает цену и добавляет
	// ключ в house_owned одним UPDATE (false — не хватает кудосов или уже
	// куплен, сервис различает по прочитанному снимку).
	BuyHouseDecor(ctx context.Context, userID int64, key string, price int) (bool, error)
	// SaveHousePlaced — узкое сохранение расстановки домика (свободные
	// координаты предметов в процентах сцены).
	SaveHousePlaced(ctx context.Context, userID int64, placed []HouseItem) error
	// SaveHouseTheme — узкое сохранение темы комнаты (ключ из HouseThemes).
	SaveHouseTheme(ctx context.Context, userID int64, theme string) error
	// SaveHousePetPos — позиция самого грувика в сцене комнаты (проценты).
	SaveHousePetPos(ctx context.Context, userID int64, x, y float64) error

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

// BankRepo — кудо-банк: леджер движений кудосов, переводы, вклад и кредит.
// Все мутации балансов — атомарные UPDATE с guard'ами в WHERE (по образцу
// BuyHouseDecor); ok=false — гейт не сошёлся (недостаток средств и т.п.),
// вызывающий различает причину по прочитанному снимку питомца.
type BankRepo interface {
	// AppendLedger — запись в выписку (вне транзакций мутаций — как журнал).
	AppendLedger(ctx context.Context, e *LedgerEntry) error
	// ListLedger — выписка пользователя, keyset по id вниз (beforeID 0 — с начала).
	ListLedger(ctx context.Context, userID, beforeID int64, limit int) ([]*LedgerEntry, error)
	// Transfer — атомарный перевод: списание с guard kudos >= amount,
	// зачисление получателю и обе записи леджера в одной транзакции.
	Transfer(ctx context.Context, fromID, toID, companyID int64, amount int, comment string) (fromKudos int, ok bool, err error)
	// MonthlyTotals — приход/расход по выписке за последние 30 дней.
	MonthlyTotals(ctx context.Context, userID int64) (in, out int, err error)
	// LifetimeEarned — суммарно заработано за всё время (без transfer_in/
	// loan_taken/bank_withdraw) — определяет уровень клиента банка.
	LifetimeEarned(ctx context.Context, userID int64) (int, error)
	// TopGenerous — топ дарителей компании за 30 дней (по transfer_out).
	TopGenerous(ctx context.Context, companyID int64, limit int) ([]GenerousEntry, error)
	// DepositSavings — кошелёк → вклад (guard: kudos >= amount, долга нет).
	DepositSavings(ctx context.Context, userID int64, amount int) (kudos, savings int, ok bool, err error)
	// WithdrawSavings — вклад → кошелёк (guard: savings >= amount); вклад,
	// сходящий в ноль, обнуляет отметку начисления процентов.
	WithdrawSavings(ctx context.Context, userID int64, amount int) (kudos, savings int, ok bool, err error)
	// AccrueSavings — ленивое начисление процентов за целые прошедшие сутки
	// (одним UPDATE + запись леджера; повторный конкурентный вызов начислит 0).
	AccrueSavings(ctx context.Context, userID, companyID int64, ratePct int) (interest int, err error)
	// TakeLoan — выдача кредита: +amount на кошелёк, долг = amount + комиссия
	// (guard: активного долга нет).
	TakeLoan(ctx context.Context, userID int64, amount, debt int) (kudos int, ok bool, err error)
	// RepayLoan — погашение с кошелька (guard: kudos >= amount и долг >= amount).
	RepayLoan(ctx context.Context, userID int64, amount int) (kudos, loan int, ok bool, err error)
	// DeleteLedger — чистка выписки при удалении питомца.
	DeleteLedger(ctx context.Context, userID int64) error

	// ── Копилки-цели ────────────────────────────────────────────────
	ListGoals(ctx context.Context, userID int64) ([]*BankGoal, error)
	// CreateGoal — новая копилка (ID проставляется в g).
	CreateGoal(ctx context.Context, g *BankGoal) error
	// GoalDeposit — кошелёк → копилка (guard: kudos >= amount, копилка своя);
	// achievedNow — цель достигнута именно этим пополнением (ровно один раз).
	GoalDeposit(ctx context.Context, userID, goalID int64, amount int) (goal *BankGoal, achievedNow, ok bool, err error)
	// GoalWithdraw — копилка → кошелёк (guard: saved >= amount).
	GoalWithdraw(ctx context.Context, userID, goalID int64, amount int) (goal *BankGoal, ok bool, err error)
	// DeleteGoal — удаление копилки с возвратом остатка в кошелёк.
	DeleteGoal(ctx context.Context, userID, goalID int64) (refund int, ok bool, err error)

	// ── Благотворительные сборы компании ────────────────────────────
	// ListFunds — активные сборы + последние finishedShown завершённых, с
	// агрегатами витрины (доноры, мой вклад) для viewerID.
	ListFunds(ctx context.Context, companyID, viewerID int64, finishedShown int) ([]*BankFund, error)
	// CreateFund — новый сбор (ID проставляется в f).
	CreateFund(ctx context.Context, f *BankFund) error
	// Donate — пожертвование: кошелёк → сбор, запись донации и леджер одной
	// транзакцией. fundOK=false — сбор не найден/не активен; ok=false — не
	// хватает кудосов; completedNow — цель закрыта именно этим взносом.
	Donate(ctx context.Context, userID, fundID, companyID int64, amount int) (fund *BankFund, fundOK, ok, completedNow bool, err error)
	// CloseFund — досрочное закрытие сбора (false — не найден/не активен).
	CloseFund(ctx context.Context, fundID, companyID int64) (bool, error)
	// FundTopDonors — топ доноров сбора.
	FundTopDonors(ctx context.Context, fundID int64, limit int) ([]GenerousEntry, error)

	// ── Статистика ──────────────────────────────────────────────────
	// DailyTotals — приход/расход по дням (МСК) за последние days дней.
	DailyTotals(ctx context.Context, userID int64, days int) ([]BankDayStat, error)
	// KindTotals — приход/расход по видам операций за последние days дней.
	KindTotals(ctx context.Context, userID int64, days int) ([]BankKindStat, error)
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
