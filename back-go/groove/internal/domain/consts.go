package domain

import "time"

// MSK — все «дневные» механики Groove считаются по московскому времени.
var MSK = time.FixedZone("MSK", 3*60*60)

// Уровень роли «Администратор» компании (удаление чужих комментариев).
const LevelAdmin = 3

// Фиксированный набор реакций — продублирован на фронте в utils/groove.js.
var FeedReactions = []string{"🔥", "💪", "👏", "🎉", "😮", "❤️"}

const FeedPageLimit = 30

// FeedExcludedKinds — машинные события старой «ленты активности»: больше не
// создаются, а исторические записи не показываются в выборке.
var FeedExcludedKinds = []string{"unit_started", "unit_stopped"}

// KudosCategories — категории благодарностей (доска признания).
// Продублированы на фронте в utils/groove.js (KUDOS_CATEGORIES).
var KudosCategories = map[string]string{
	"helped":  "Выручил",
	"quality": "Качество",
	"speed":   "Скорость",
}

// Накопительные пороги XP для стадий: яйцо → малыш → непоседа → подросток
// → взрослый → герой → легенда.
var StageXP = []int{0, 40, 120, 280, 550, 950, 1500}

var MaxStage = len(StageXP) - 1

const (
	FeedCost     = 3
	FeedXP       = 12
	FeedDailyMax = 6
)

// ── Прямой XP за работу ────────────────────────────────────────────
// Работа растит питомца напрямую, а не только через «паёк» кормлений:
// юниты и закрытые задачи дают XP с дневными капами; кормление в тот же
// день включает «сытость» — прямой XP умножается на FedXPBoost.

const (
	XPUnitMinutesPer = 3   // 1 XP за каждые 3 минуты юнита
	XPUnitDailyCap   = 40  // дневной кап XP от юнитов
	XPTaskClosed     = 8   // XP за закрытую задачу
	XPTaskDailyCap   = 40  // дневной кап XP от закрытых задач
	FedXPBoost       = 1.5 // множитель прямого XP, пока питомец сегодня кормлен
)

// Дневные капы грувов по источникам (неизвестный источник — кап 10).
var DailyCaps = map[string]int{
	"unit":        15, // завершённые юниты
	"task_closed": 25, // закрытые задачи
	"reaction":    10, // полученные реакции
	"kudos":       10, // полученные благодарности
}

const DefaultDailyCap = 10

// Магазин аксессуаров (эмодзи-маппинг — на фронте, utils/groove.js).
var ShopPrices = map[string]int{
	"party": 30, "cap": 40, "bow": 40, "scarf": 50, "tie": 50,
	"glasses": 60, "headphones": 60, "mask": 70, "tophat": 80,
	"medal": 90, "crown": 200,
}

const (
	RaidRewardItem = "helmet" // только за победу в рейде, не продаётся
	RaidWinBeans   = 15
)

var StreakMilestones = map[int]bool{
	3: true, 5: true, 7: true, 10: true, 14: true,
	21: true, 30: true, 50: true, 100: true,
}

var Bosses = []string{"Дедлайнозавр", "Багоблин", "Прокрастинатор",
	"Совещаниус", "Хаос-гоблин", "Технодолг"}

// ── Ежедневный квест ───────────────────────────────────────────────

const QuestRewardBeans = 20

type QuestTemplate struct {
	Kind   string
	Target int
	Title  string
	Unit   string
	Hint   string
}

var QuestTemplates = []QuestTemplate{
	{"tasks_closed", 2, "Закрыть 2 задачи", "задач",
		"Две задачи в архив. Счётчик рейда тоже заметит."},
	{"tasks_closed", 3, "Закрыть 3 задачи", "задач",
		"Три закрытия за день. Выполнимо, проверено статистикой."},
	{"units_finished", 3, "Завершить 3 юнита", "юнитов",
		"Три подхода любой длины. По 25–50 минут — рабочая схема."},
	{"unit_minutes", 60, "60 минут в фокусе", "мин",
		"Час чистой работы. Грувик обещает не отвлекать."},
	{"unit_minutes", 90, "Полтора часа фокуса", "мин",
		"90 минут суммарно. Один большой юнит или несколько — без разницы."},
	{"feed_pet", 1, "Покормить Грувика", "раз",
		"Одно кормление: 12 XP плюс сытость — прямой XP за работу ×1.5 до конца дня."},
}

// ── Болезнь ────────────────────────────────────────────────────────
// Грувик заболевает, если хозяин SickAfterDays РАБОЧИХ дней не завершал
// юниты. Лечение — recovery-очки: работа и «бульон».

const (
	SickAfterDays          = 5
	RecoveryTarget         = 3
	SickFeedCost           = 1
	SickFeedDailyMax       = 2
	RecoveryMinUnitMinutes = 15
)

// ── Характер ───────────────────────────────────────────────────────

type Personality struct {
	Title string
	Hint  string
}

var Personalities = map[string]Personality{
	"lazy":      {"Ленивец-мечтатель", "работает редко, любит подремать и пофилософствовать"},
	"night":     {"Ночной активист", "оживает после заката, ночь — его стихия"},
	"early":     {"Ранняя пташка", "лучшие дела делает до обеда, бодрится с утра"},
	"energizer": {"Бодрячок-энерджайзер", "куча коротких подходов, энергия бьёт ключом"},
	"zen":       {"Дзен-марафонец", "длинные сосредоточенные сессии, спокоен как удав"},
	"steady":    {"Уравновешенный трудяга", "ровный стабильный ритм, надёжен и рассудителен"},
}

// Русские названия для AI-промптов (стадии ≡ фронтовым PET_STAGES).
var PetStagesTitles = []string{"Яйцо", "Малыш", "Непоседа", "Подросток",
	"Взрослый", "Герой", "Легенда"}

var PetSpeciesTitles = map[string]string{
	"egg": "ещё не вылупившийся", "owl": "сова", "lark": "жаворонок",
	"sprinter": "спринтер", "marathoner": "марафонец", "fox": "лис-универсал",
	"cat": "котёнок", "dog": "щенок", "tiger": "тигрёнок", "bear": "медвежонок",
	"rabbit": "крольчонок", "frog": "лягушонок", "panda": "панда",
	"penguin": "пингвинёнок", "monkey": "обезьянка", "chick": "цыплёнок",
	"hamster": "хомячок", "hedgehog": "ёжик", "koala": "коала", "deer": "оленёнок",
	"bee": "пчёлка", "octopus": "осьминожка", "wolf": "волчонок", "lion": "львёнок",
	"dolphin": "дельфин", "whale": "китёнок",
	"unicorn": "единорог", "dragon": "дракон",
}

// Магазин «видов» Грувика; природные виды бесплатны и приходят с эволюцией.
var SpeciesShop = map[string]int{
	"cat": 80, "dog": 80, "rabbit": 80, "frog": 80,
	"chick": 100, "monkey": 100, "panda": 120,
	"tiger": 140, "bear": 140, "penguin": 140,
	"hamster": 90, "hedgehog": 110, "koala": 130, "deer": 150,
	"bee": 160, "octopus": 170, "wolf": 180, "lion": 200,
	"dolphin": 200, "whale": 230,
	"unicorn": 250, "dragon": 250,
}

var NaturalSpecies = map[string]bool{
	"owl": true, "lark": true, "sprinter": true, "marathoner": true, "fox": true,
}

// ── Сезонные товары ────────────────────────────────────────────────
// Несколько аксессуаров на каждый сезон; купить можно только в свой сезон.

var SeasonalItems = map[string]int{
	"santa": 45, "snowman": 45, "mittens": 45,
	"flower": 45, "butterfly": 45, "rainbow": 45,
	"icecream": 45, "sunhat": 45, "watermelon": 45,
	"pumpkin": 45, "leaf": 45, "mushroom": 45,
}

type Season struct {
	Title string
	Items []string
}

var (
	winterItems = []string{"santa", "snowman", "mittens"}
	springItems = []string{"flower", "butterfly", "rainbow"}
	summerItems = []string{"icecream", "sunhat", "watermelon"}
	autumnItems = []string{"pumpkin", "leaf", "mushroom"}
)

var SeasonByMonth = map[time.Month]Season{
	time.December: {"Зима", winterItems}, time.January: {"Зима", winterItems}, time.February: {"Зима", winterItems},
	time.March: {"Весна", springItems}, time.April: {"Весна", springItems}, time.May: {"Весна", springItems},
	time.June: {"Лето", summerItems}, time.July: {"Лето", summerItems}, time.August: {"Лето", summerItems},
	time.September: {"Осень", autumnItems}, time.October: {"Осень", autumnItems}, time.November: {"Осень", autumnItems},
}

// ── Редкие праздничные товары ──────────────────────────────────────
// Уникальные аксессуары, доступные только в короткое окно вокруг своего
// праздника; вне окна их не купить (вернутся к следующему событию).

type DateWindow struct {
	FromMonth time.Month
	FromDay   int
	ToMonth   time.Month
	ToDay     int
}

type RareItem struct {
	Price  int
	Window DateWindow
}

var RareItems = map[string]RareItem{
	"fireworks":  {80, DateWindow{time.December, 25, time.January, 8}},
	"love":       {60, DateWindow{time.February, 10, time.February, 16}},
	"shamrock":   {60, DateWindow{time.March, 14, time.March, 20}},
	"rocket":     {70, DateWindow{time.April, 8, time.April, 16}},
	"graduation": {75, DateWindow{time.June, 20, time.July, 10}},
}

// InDateWindow — попадает ли момент now в окно w (с переносом через Новый год).
func InDateWindow(now time.Time, w DateWindow) bool {
	cur := int(now.Month())*100 + now.Day()
	from := int(w.FromMonth)*100 + w.FromDay
	to := int(w.ToMonth)*100 + w.ToDay
	if from <= to {
		return cur >= from && cur <= to
	}
	return cur >= from || cur <= to
}

// DefaultWeekend — дефолтные выходные (суббота, воскресенье; 0=Пн … 6=Вс).
var DefaultWeekend = []int{5, 6}
