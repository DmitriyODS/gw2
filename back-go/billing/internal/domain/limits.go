package domain

import "time"

// Линейка тарифов. ЛИМИТЫ конечны и живут здесь (менять их — правка кода и
// выката), а ЦЕНЫ, названия и подводки правит супер-админ в разделе «Аудит
// платформы» — они в таблице billing_plans.
//
// Скоуп лимита определяется его смыслом:
//   - личные (хранилище, токены, ежедневники, доски, папки чатов, статусы,
//     премиум-контент, число своих компаний) — по тарифу самого пользователя;
//   - компанийные (участники, задачи, календари, реестры, портал, расширенная
//     статистика, экспорт данных) — по тарифу СОЗДАТЕЛЯ компании.

// Unlimited — «без ограничения».
const Unlimited = -1

/* Подписки и оплата не успели к выпуску 7.0 и СКРЫТЫ от пользователя: витрины,
   тарифов и счетов он не видит. Механика ниже остаётся целой и вернётся вместе
   с оплатой, но пока платить не за что — токены ИИ выдаются всем одинаково и не
   зависят от тарифа: DailyAITokens в сутки, период тоже сутки (по МСК, как и
   остальная статистика платформы).

   Вернуть тарифную квоту = снять SubscriptionsHidden: AIQuota снова начнёт
   отдавать Limits.AITokens и месячный период. */

const (
	// SubscriptionsHidden — подписки скрыты от пользователя (см. выше).
	SubscriptionsHidden = true
	// DailyAITokens — сколько токенов доступа получает КАЖДЫЙ пользователь в сутки.
	DailyAITokens int64 = 1000
	// FreeStorageBytes — хранилище, доступное КАЖДОМУ, пока подписки скрыты.
	// В отличие от прочих счётных лимитов место снять нельзя: оно упирается в
	// физический диск, а не в тариф. Раздел «Настройки → Хранилище» показывает
	// расход и даёт освободить место, когда оно кончилось.
	FreeStorageBytes = 5 * GiB
)

// MoscowOffset — смещение МСК: сутки квоты кончаются в московскую полночь.
const MoscowOffset = 3 * time.Hour

/* EffectiveLimits — лимиты, которые реально применяются к пользователю.

   Пока подписки скрыты, тариф ограничивать НЕ ДОЛЖЕН: заплатить всё равно
   нельзя, а бесплатный «Джун» разрешает одну доску, один календарь и трёх
   человек в компании — упёршемуся некуда идти. Поэтому счётные лимиты
   становятся безграничными, а платные возможности — доступными.

   Держатся два ограничения, за которыми стоят живые деньги, а не тариф:
   токены ИИ (их выдаёт AIQuota) и место в хранилище — FreeStorageBytes всем
   поровну. Докупить место сейчас нельзя, поэтому упереться в него можно;
   выход — раздел «Настройки → Хранилище», где видно, чем оно занято.

   Тарифная линейка при этом цела: PlanLimits не тронут, и снятие
   SubscriptionsHidden возвращает всё как было. */
func EffectiveLimits(l Limits) Limits {
	if !SubscriptionsHidden {
		return l
	}
	return Limits{
		Tasks:            Unlimited,
		Companies:        Unlimited,
		Members:          Unlimited,
		StorageBytes:     FreeStorageBytes,
		AITokens:         l.AITokens, // не участвует: квоту считает AIQuota
		Calendars:        Unlimited,
		Diaries:          Unlimited,
		Boards:           Unlimited,
		Registries:       Unlimited,
		ChatFolders:      Unlimited,
		CallParticipants: Unlimited,

		DataTransfer:    true,
		AdvancedStats:   true,
		Portal:          true,
		UserStatuses:    true,
		PremiumThemes:   true,
		PremiumPetSkins: true,
		PremiumPetHouse: true,
		PremiumPetGoods: true,
	}
}

// AIQuota — сколько токенов выдано пользователю и когда период закончится.
// Пока подписки скрыты, тариф на это не влияет.
func AIQuota(limits Limits, now time.Time) (tokens int64, periodEnd time.Time) {
	if SubscriptionsHidden {
		return DailyAITokens, nextMoscowMidnight(now)
	}
	return max(limits.AITokens, 0), now.AddDate(0, 1, 0)
}

// nextMoscowMidnight — ближайшая полночь по Москве после now.
func nextMoscowMidnight(now time.Time) time.Time {
	msk := now.UTC().Add(MoscowOffset)
	midnight := time.Date(msk.Year(), msk.Month(), msk.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
	return midnight.Add(-MoscowOffset)
}

// Коды тарифов.
const (
	PlanJunior = "junior"
	PlanMiddle = "middle"
	PlanSenior = "senior"
)

// PlanCodes — порядок линейки от младшего к старшему (он же порядок витрины).
var PlanCodes = []string{PlanJunior, PlanMiddle, PlanSenior}

// Гигабайт в байтах — хранилище задаётся в гигабайтах.
const GiB int64 = 1 << 30

// Limits — что тариф разрешает. Числовое поле == Unlimited — без ограничения.
type Limits struct {
	Tasks            int64 `json:"tasks"`             // задач в компании
	Companies        int   `json:"companies"`         // своих компаний у пользователя
	Members          int   `json:"members"`           // человек в компании
	StorageBytes     int64 `json:"storage_bytes"`     // личное хранилище
	AITokens         int64 `json:"ai_tokens"`         // токенов доступа в месяц
	Calendars        int   `json:"calendars"`         // календарей в компании
	Diaries          int   `json:"diaries"`           // личных ежедневников
	Boards           int   `json:"boards"`            // личных досок
	Registries       int   `json:"registries"`        // реестров в компании
	ChatFolders      int   `json:"chat_folders"`      // папок чатов
	CallParticipants int   `json:"call_participants"` // человек в групповом звонке

	DataTransfer    bool `json:"data_transfer"`     // экспорт и импорт данных компании
	AdvancedStats   bool `json:"advanced_stats"`    // расширенная статистика компании
	Portal          bool `json:"portal"`            // корпоративный портал
	UserStatuses    bool `json:"user_statuses"`     // пользовательские статусы
	PremiumThemes   bool `json:"premium_themes"`    // премиум-темы
	PremiumPetSkins bool `json:"premium_pet_skins"` // премиум-скины питомцев
	PremiumPetHouse bool `json:"premium_pet_house"` // премиум-скины домика
	PremiumPetGoods bool `json:"premium_pet_goods"` // премиум-товары питомцам
}

// PlanLimits — линейка «Джун / Мидл / Синьор».
var PlanLimits = map[string]Limits{
	PlanJunior: {
		Tasks:            800,
		Companies:        1,
		Members:          3,
		StorageBytes:     5 * GiB,
		AITokens:         0,
		Calendars:        1,
		Diaries:          1,
		Boards:           1,
		Registries:       1,
		ChatFolders:      0,
		CallParticipants: 3,
	},
	PlanMiddle: {
		Tasks:            Unlimited,
		Companies:        3,
		Members:          10,
		StorageBytes:     10 * GiB,
		AITokens:         1000,
		Calendars:        5,
		Diaries:          5,
		Boards:           5,
		Registries:       5,
		ChatFolders:      10,
		CallParticipants: 5,
		DataTransfer:     true,
		AdvancedStats:    true,
		Portal:           true,
		UserStatuses:     true,
	},
	PlanSenior: {
		Tasks:            Unlimited,
		Companies:        5,
		Members:          15,
		StorageBytes:     20 * GiB,
		AITokens:         10000,
		Calendars:        Unlimited,
		Diaries:          Unlimited,
		Boards:           Unlimited,
		Registries:       Unlimited,
		ChatFolders:      Unlimited,
		CallParticipants: 10,
		DataTransfer:     true,
		AdvancedStats:    true,
		Portal:           true,
		UserStatuses:     true,
		PremiumThemes:    true,
		PremiumPetSkins:  true,
		PremiumPetHouse:  true,
		PremiumPetGoods:  true,
	},
}

// LimitsFor — лимиты тарифа; неизвестный код трактуем как бесплатный (тариф
// могли выключить или переименовать, но доступ терять нельзя).
func LimitsFor(plan string) Limits {
	if l, ok := PlanLimits[plan]; ok {
		return l
	}
	return PlanLimits[PlanJunior]
}

// PlanRank — место тарифа в линейке (для сравнения «выше/ниже»).
func PlanRank(plan string) int {
	for i, code := range PlanCodes {
		if code == plan {
			return i
		}
	}
	return 0
}

// AddAmount — прибавить к числовому лимиту докупленное. Безлимит остаётся
// безлимитом, из нуля докупка делает конечный лимит.
func AddAmount(limit int, extra int) int {
	if limit == Unlimited {
		return Unlimited
	}
	return limit + extra
}

// AddAmount64 — то же для больших величин (байты, токены).
func AddAmount64(limit int64, extra int64) int64 {
	if limit == Unlimited {
		return Unlimited
	}
	return limit + extra
}

// Allows — влезает ли ещё одна сущность при текущем количестве current.
func Allows(limit int, current int) bool {
	return limit == Unlimited || current < limit
}
