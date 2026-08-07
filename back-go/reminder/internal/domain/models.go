package domain

import "time"

// Reminder — личное напоминание пользователя: сработает в RemindAt и, если
// задан повтор, встанет на следующий срок. Принадлежит ОДНОМУ пользователю
// (OwnerID) и не зависит от компании (кросс-компанийное, как ежедневник).
//
// Все времена в БД — UTC; Timezone (IANA) нужен повторам: «каждый день в 9:00»
// обязано оставаться девятью утра пользователя и после перевода часов.
type Reminder struct {
	ID      int64  `json:"id"`
	OwnerID int64  `json:"owner_id"`
	Title   string `json:"title"`
	Note    string `json:"note"`
	// RemindAt — ближайшее (или последнее для завершённых) срабатывание.
	RemindAt time.Time `json:"remind_at"`
	Timezone string    `json:"timezone"`
	Repeat   Repeat    `json:"repeat"`
	Link     Link      `json:"link"`
	// Active — напоминание ждёт своего часа. Разовое после срабатывания
	// становится неактивным (уходит в «Сработавшие»), повторяющееся — нет.
	Active bool `json:"active"`
	// LastFiredAt — когда сработало в последний раз (nil — ещё ни разу).
	LastFiredAt *time.Time `json:"last_fired_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Виды повтора. weekdays — по рабочим дням (пн–пт), weekly с Days — по
// выбранным дням недели.
const (
	RepeatNone     = "none"
	RepeatDaily    = "daily"
	RepeatWeekdays = "weekdays"
	RepeatWeekly   = "weekly"
	RepeatMonthly  = "monthly"
	RepeatYearly   = "yearly"
)

var RepeatKinds = map[string]bool{
	RepeatNone: true, RepeatDaily: true, RepeatWeekdays: true,
	RepeatWeekly: true, RepeatMonthly: true, RepeatYearly: true,
}

// Repeat — правило повторения. Interval — «каждые N» единиц вида (1 — каждый
// день/неделю/месяц/год). Days — дни недели для weekly (1 — понедельник,
// 7 — воскресенье). Until — дата, после которой повторов больше нет.
type Repeat struct {
	Kind     string     `json:"kind"`
	Interval int        `json:"interval"`
	Days     []int      `json:"days"`
	Until    *time.Time `json:"until"`
}

// Виды привязки напоминания к записи другого раздела.
const (
	LinkNone     = "none"
	LinkDiary    = "diary"
	LinkCalendar = "calendar"
)

var LinkKinds = map[string]bool{LinkNone: true, LinkDiary: true, LinkCalendar: true}

// Link — привязка к записи ежедневника или календаря: напоминание хранит
// СНИМОК (время события уже учтено в RemindAt, название — для карточки) и
// ссылку для перехода. Владелец записи и напоминания — один и тот же человек,
// поэтому чужих данных здесь нет; актуализирует снимок тот раздел, который
// правит запись (см. front/src/stores/reminders.js).
type Link struct {
	Kind string `json:"kind"`
	// ParentID — id ежедневника/календаря (нужен для открытия раздела).
	ParentID int64 `json:"parent_id"`
	// RecordID — id самой записи.
	RecordID int64 `json:"record_id"`
	// Title — название записи на момент привязки (подпись в карточке).
	Title string `json:"title"`
	// LeadMinutes — за сколько минут до события напомнить (0 — в момент).
	LeadMinutes int `json:"lead_minutes"`
}

// ListScope — какие напоминания показывать.
type ListScope string

const (
	// ScopeActive — ждущие своего часа (по возрастанию времени).
	ScopeActive ListScope = "active"
	// ScopeDone — уже сработавшие разовые (журнал, по убыванию времени).
	ScopeDone ListScope = "done"
	// ScopeAll — всё сразу.
	ScopeAll ListScope = "all"
)

// ReminderUpdate — частичная правка: nil-поля не меняются.
type ReminderUpdate struct {
	Title    *string
	Note     *string
	RemindAt *time.Time
	Timezone *string
	Repeat   *Repeat
	Link     *Link
	Active   *bool
}

// User — идентичность пользователя для авторизации.
type User struct {
	ID           int64
	FIO          string
	AvatarPath   *string
	IsActive     bool
	IsSuperAdmin bool
}
