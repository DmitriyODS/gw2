package domain

import "time"

// ── Потребности грувика ─────────────────────────────────────────────
// Четыре шкалы 0..100, тающие со временем. Пересчёт ЛЕНИВЫЙ (по needs_at при
// чтении/действии — как возврат из приключения, без фонового цикла), поэтому
// убывание задано целыми единицами за фиксированный тик: дробный остаток
// времени не сгорал бы при частом поллинге клиента, а needs_at сдвигается
// ровно на применённые тики.
//
// Три шкалы имеют «свою» болезнь и вводят в неё, опустев (Need.Ailment);
// общение болезни не даёт — оно кормит настроение, а настроение множит XP.

const (
	NeedMax  = 100
	NeedTick = 30 * time.Minute
)

// Ключи шкал (≡ NEEDS на фронте, front/src/utils/pets.js).
const (
	NeedSatiety = "satiety"
	NeedEnergy  = "energy"
	NeedHygiene = "hygiene"
	NeedSocial  = "social"
)

// Need — описание шкалы: скорость убывания и болезнь пустой шкалы.
type Need struct {
	Key          string
	Title        string
	DecayPerTick int
	Ailment      string // "" — пустая шкала болезни не даёт
}

// Needs — порядок = порядок показа шкал у клиента.
var Needs = []Need{
	{NeedSatiety, "Сытость", 2, AilmentHunger}, // сутки без еды — истощение
	{NeedEnergy, "Энергия", 1, AilmentCold},    // ~двое суток без сна — простуда
	{NeedHygiene, "Чистота", 1, AilmentGrime},
	{NeedSocial, "Общение", 1, ""},
}

// NeedValues — значения шкал питомца.
type NeedValues struct {
	Satiety int `json:"satiety"`
	Energy  int `json:"energy"`
	Hygiene int `json:"hygiene"`
	Social  int `json:"social"`
}

func (n *NeedValues) ptr(key string) *int {
	switch key {
	case NeedSatiety:
		return &n.Satiety
	case NeedEnergy:
		return &n.Energy
	case NeedHygiene:
		return &n.Hygiene
	case NeedSocial:
		return &n.Social
	}
	return nil
}

// Get — значение шкалы (неизвестный ключ — 0).
func (n *NeedValues) Get(key string) int {
	if p := n.ptr(key); p != nil {
		return *p
	}
	return 0
}

// Add — изменить шкалу с клампом в 0..NeedMax.
func (n *NeedValues) Add(key string, delta int) {
	p := n.ptr(key)
	if p == nil {
		return
	}
	*p = min(NeedMax, max(0, *p+delta))
}

// Mood — настроение: среднее по шкалам, где самая запущенная потребность
// весит втрое (иначе одна брошенная шкала из четырёх почти не портила бы
// настроение — а грувик, которого не кормят, доволен быть не может).
// Множит прямой XP за работу (MoodFactor) — потребности не декорация, а
// экономика.
func (n *NeedValues) Mood() int {
	worst := min(n.Satiety, min(n.Energy, min(n.Hygiene, n.Social)))
	return (n.Satiety + n.Energy + n.Hygiene + n.Social + 2*worst) / 6
}

// MoodFactor — множитель прямого XP за работу по настроению. Ухоженный
// питомец учится в полтора раза быстрее, запущенный — заметно медленнее.
func MoodFactor(mood int) float64 {
	switch {
	case mood >= 80:
		return 1.5
	case mood >= 60:
		return 1.2
	case mood >= 40:
		return 1.0
	case mood >= 20:
		return 0.85
	}
	return 0.7
}

// MoodTitle — подпись настроения (≡ moodTitle на фронте).
func MoodTitle(mood int) string {
	switch {
	case mood >= 80:
		return "Отличное"
	case mood >= 60:
		return "Хорошее"
	case mood >= 40:
		return "Обычное"
	case mood >= 20:
		return "Так себе"
	}
	return "Плохое"
}

// ── Действия и их влияние на потребности ────────────────────────────
// Ключи действий общие для NeedGains и Ailment.Cures.

const (
	ActionFeed  = "feed"
	ActionWalk  = "walk"
	ActionHeal  = "heal"
	ActionSleep = "sleep"
	ActionBath  = "bath"
	ActionWork  = "work" // юниты/закрытые задачи (хуки tasksvc)
	// Поглаживание с двух сторон: получателю внимания достаётся больше, чем
	// тому, кто гладит сам.
	ActionStrokeIn  = "stroke_in"
	ActionStrokeOut = "stroke_out"
)

// NeedGains — как действие двигает шкалы. Отрицательные значения намеренны:
// прогулка бодрит общением, но пачкает и утомляет — потребности связаны, и
// уход за питомцем становится циклом, а не одной кнопкой.
var NeedGains = map[string]map[string]int{
	ActionFeed:      {NeedSatiety: 40, NeedHygiene: -5},
	ActionWalk:      {NeedSocial: 15, NeedEnergy: -10, NeedHygiene: -10, NeedSatiety: -5},
	ActionSleep:     {NeedEnergy: 55},
	ActionBath:      {NeedHygiene: 70, NeedEnergy: -5},
	ActionStrokeIn:  {NeedSocial: 25},
	ActionStrokeOut: {NeedSocial: 10},
	ActionWork:      {NeedEnergy: -3, NeedSatiety: -3},
}

// ── Сон и купание ───────────────────────────────────────────────────
// Сон бесплатен (энергия не должна быть платной стеной), купание — платное.

const (
	SleepDailyMax = 2
	BathCost      = 12
	BathDailyMax  = 3
)

// ── Болезни ─────────────────────────────────────────────────────────
// Раньше болезнь была одна (простой в работе). Теперь у каждой запущенной
// потребности своя болезнь со СВОИМ рецептом: неверное лечение почти не
// помогает, верное поднимает питомца за одно-два действия. Общий счётчик —
// RecoveryTarget очков (pets.recovery), вид болезни — pets.ailment.

const (
	AilmentBlues  = "blues"  // хандра: простой в работе (историческая механика)
	AilmentHunger = "hunger" // истощение: пустая сытость
	AilmentCold   = "cold"   // простуда: пустая энергия
	AilmentGrime  = "grime"  // грязнуля: пустая чистота
)

// Ailment — болезнь: причина, подпись и рецепт (действие → очки выздоровления).
type Ailment struct {
	Key   string
	Title string
	Hint  string
	Cures map[string]int
}

// Ailments — каталог болезней (≡ AILMENTS на фронте, front/src/utils/pets.js).
var Ailments = map[string]Ailment{
	AilmentBlues: {
		Key: AilmentBlues, Title: "Хандра",
		Hint:  "Загрустил без работы. Лечится юнитами, прогулкой и заботой.",
		Cures: map[string]int{ActionWork: 1, ActionWalk: 1, ActionHeal: 1},
	},
	AilmentHunger: {
		Key: AilmentHunger, Title: "Истощение",
		Hint:  "Слишком долго не ел. Лечится едой — бульон вернёт в строй.",
		Cures: map[string]int{ActionFeed: 2, ActionSleep: 1, ActionHeal: 1},
	},
	AilmentCold: {
		Key: AilmentCold, Title: "Простуда",
		Hint:  "Выдохся и слёг. Лечится сном и аптечкой, еда помогает меньше.",
		Cures: map[string]int{ActionSleep: 2, ActionHeal: 2, ActionFeed: 1},
	},
	AilmentGrime: {
		Key: AilmentGrime, Title: "Грязнуля",
		Hint:  "Зарос грязью и чешется. Лечится купанием — одного раза хватит.",
		Cures: map[string]int{ActionBath: 3, ActionHeal: 1},
	},
}

// CureFor — очки выздоровления, которые действие даёт при этой болезни
// (0 — рецепт не тот: действие сделано, но питомцу оно не помогло).
func CureFor(ailment, action string) int {
	a, ok := Ailments[ailment]
	if !ok {
		return 0
	}
	return a.Cures[action]
}

// AilmentTitle — подпись болезни (неизвестный ключ — общая «Болезнь»).
func AilmentTitle(ailment string) string {
	if a, ok := Ailments[ailment]; ok {
		return a.Title
	}
	return "Болезнь"
}
