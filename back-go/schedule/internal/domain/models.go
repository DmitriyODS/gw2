package domain

import (
	"encoding/json"
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// DateLayout — формат даты без времени: якорь цикла и границы своего правила
// повтора — это дни, а не моменты.
const DateLayout = "2006-01-02"

// Типы полей карточки занятия — общий набор pkg/records БЕЗ файловых:
// расписание файлов не держит вовсе, поэтому у него нет ни квоты, ни чанковой
// загрузки, ни контракта владельца файлов для «Хранилища».
const (
	FieldText     = records.FieldText
	FieldTextarea = records.FieldTextarea
	FieldNumber   = records.FieldNumber
	FieldRegex    = records.FieldRegex
	FieldPhone    = records.FieldPhone
	FieldEmail    = records.FieldEmail
	FieldCheckbox = records.FieldCheckbox
	FieldSelect   = records.FieldSelect
	FieldLink     = records.FieldLink
	FieldDatetime = records.FieldDatetime
)

// FieldTypes — допустимые типы поля занятия.
var FieldTypes = map[string]bool{
	FieldText: true, FieldTextarea: true, FieldNumber: true, FieldRegex: true,
	FieldPhone: true, FieldEmail: true, FieldCheckbox: true, FieldSelect: true,
	FieldLink: true, FieldDatetime: true,
}

// GridCols — ширина сетки карточки занятия в колонках.
const GridCols = 2

// Schedule — расписание: регулярная сетка занятий с циклом в CycleWeeks недель.
// Принадлежит ОДНОМУ человеку и не зависит от компании; другим доступно только
// на чтение — публичной ссылкой или адресно.
// Расписаний у человека много, и они независимы: своя цикличность, свои
// категории, свои поля. Цикл живёт ВНУТРИ расписания и к их списку отношения
// не имеет.
type Schedule struct {
	ID      int64  `json:"id"`
	OwnerID int64  `json:"owner_id"`
	Name    string `json:"name"`
	// CycleWeeks — длина цикла в неделях (1..MaxCycleWeeks). 1 — обычная
	// неделя, 2 — числитель/знаменатель.
	CycleWeeks int `json:"cycle_weeks"`
	// CycleAnchor — понедельник недели №1 цикла: от него считается номер недели
	// для любой даты в обе стороны.
	CycleAnchor time.Time `json:"-"`
	// WeekLabels — названия недель цикла («Числитель», «Знаменатель»).
	// Пустая строка на позиции означает «Неделя N».
	WeekLabels []string `json:"week_labels"`
	// Timezone (IANA) — зона расписания: по ней считается «сегодня» и «сейчас».
	// Расписание привязано к месту (учёба, работа), а не к устройству.
	Timezone string `json:"timezone"`
	// GapMin — от скольки минут промежуток между занятиями считается окном.
	// Это умолчание расписания; на экране каждый подкручивает его себе сам.
	GapMin    int       `json:"gap_min"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Owner*/Shared заполняются только для чужих расписаний (вкладка «Поделились»).
	OwnerName   string  `json:"owner_name,omitempty"`
	OwnerAvatar *string `json:"owner_avatar,omitempty"`
	Shared      bool    `json:"shared"`
	// ItemCount — сколько занятий в расписании (подпись в списке).
	ItemCount int `json:"item_count"`

	// Categories/Fields — заполняются при чтении одного расписания и списка.
	// Без omitempty: пустой набор обязан приезжать как [], иначе на клиенте
	// получается undefined вместо массива.
	Categories []*Category `json:"categories"`
	Fields     []*Field    `json:"fields"`
}

// MarshalJSON — якорь цикла отдаётся датой YYYY-MM-DD: время и зона у него
// смысла не имеют, а клиент в другом поясе сдвинул бы его через границу суток.
func (s Schedule) MarshalJSON() ([]byte, error) {
	type alias Schedule
	return json.Marshal(struct {
		alias
		CycleAnchor string `json:"cycle_anchor"`
	}{alias(s), s.CycleAnchor.Format(DateLayout)})
}

// Category — категория занятий расписания: она же раскраска блока на шкале и
// чип-фильтр над ним. Справочник у каждого расписания СВОЙ.
type Category struct {
	ID         int64     `json:"id"`
	ScheduleID int64     `json:"schedule_id"`
	Name       string    `json:"name"`
	Color      string    `json:"color"` // ключ палитры --tag-* (см. records/TagColors)
	Position   int       `json:"position"`
	CreatedAt  time.Time `json:"created_at"`
}

// Field — дополнительное поле карточки занятия (преподаватель, аудитория,
// ссылка на встречу). Config хранит настройки типа — как в реестрах и
// календарях. ShowOnBlock выносит значение прямо на блок шкалы, ShowInCard —
// показывает в карточке занятия.
type Field struct {
	ID          int64          `json:"id"`
	ScheduleID  int64          `json:"schedule_id"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Position    int            `json:"position"`
	ColSpan     int            `json:"col_span"`
	RowSpan     int            `json:"row_span"`
	ShowOnBlock bool           `json:"show_on_block"`
	ShowInCard  bool           `json:"show_in_card"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Normalize — привести span'ы к границам сетки карточки.
func (f *Field) Normalize() { records.NormalizeSpans(&f.ColSpan, &f.RowSpan, &f.Config, GridCols) }

// Item — занятие: день недели, время в минутах от полуночи и правило повтора.
// Конкретных вхождений на даты не существует — расписание регулярное, а какая
// неделя сейчас, считается из даты (см. cycle.go).
type Item struct {
	ID         int64 `json:"id"`
	ScheduleID int64 `json:"schedule_id"`
	// Weekday — 0 понедельник … 6 воскресенье.
	Weekday  int    `json:"weekday"`
	StartMin int    `json:"start_min"`
	EndMin   int    `json:"end_min"`
	Title    string `json:"title"`
	// Short — короткое имя для узких колонок недели и телефона.
	Short      string `json:"short"`
	CategoryID *int64 `json:"category_id"`
	// Weeks — номера недель цикла (1..CycleWeeks), в которые занятие идёт.
	// Пусто — каждую неделю.
	Weeks []int `json:"weeks"`
	// RepeatEvery/RepeatFrom/RepeatUntil — своё правило повтора занятия
	// («каждые N недель с такой-то»), живущее мимо цикла расписания.
	// Заполнено ЛИБО оно, либо Weeks — см. Item.Validate.
	RepeatEvery *int       `json:"repeat_every"`
	RepeatFrom  *time.Time `json:"-"`
	RepeatUntil *time.Time `json:"-"`
	// Data — значения дополнительных полей по строковому id поля.
	Data      map[string]any `json:"data"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// MarshalJSON — границы своего правила отдаются датами YYYY-MM-DD.
func (i Item) MarshalJSON() ([]byte, error) {
	type alias Item
	return json.Marshal(struct {
		alias
		RepeatFrom  *string `json:"repeat_from"`
		RepeatUntil *string `json:"repeat_until"`
	}{alias(i), dateString(i.RepeatFrom), dateString(i.RepeatUntil)})
}

// dateString — дата дня для JSON (nil остаётся null).
func dateString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(DateLayout)
	return &s
}

// ItemScope — занятие вместе с расписанием: глобальный поиск Hola показывает,
// где именно оно найдено.
type ItemScope struct {
	ScheduleID   int64  `json:"schedule_id"`
	ScheduleName string `json:"schedule_name"`
	ItemID       int64  `json:"item_id"`
	Title        string `json:"title"`
	Weekday      int    `json:"weekday"`
	StartMin     int    `json:"start_min"`
	EndMin       int    `json:"end_min"`
	Category     string `json:"category,omitempty"`
}

// AgendaItem — занятие ближайшего времени для живой плитки рабочего стола.
type AgendaItem struct {
	ScheduleID   int64  `json:"schedule_id"`
	ScheduleName string `json:"schedule_name"`
	ItemID       int64  `json:"item_id"`
	Title        string `json:"title"`
	StartMin     int    `json:"start_min"`
	EndMin       int    `json:"end_min"`
	Color        string `json:"color,omitempty"`
}

// Agenda — сводка «прямо сейчас» по всем доступным расписаниям: что идёт, что
// дальше и сколько сегодня занято. Считает СЕРВЕР: клиент плитки правил
// повтора не знает.
type Agenda struct {
	// Now — занятие, идущее прямо сейчас (nil — сейчас окно или день кончился).
	Now *AgendaItem `json:"now"`
	// Next — ближайшее следующее занятие сегодня.
	Next *AgendaItem `json:"next"`
	// BusyMin/GapMin — сколько сегодня занято и сколько времени в окнах.
	BusyMin int `json:"busy_min"`
	GapMin  int `json:"gap_min"`
	// Total — сколько занятий сегодня всего.
	Total int `json:"total"`
}

// Share — публичная ссылка на расписание (read-only, без авторизации).
// Code в URL — capability.
type Share struct {
	ID         int64     `json:"id"`
	ScheduleID int64     `json:"schedule_id"`
	Code       string    `json:"code"`
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// UserShare — адресный доступ к расписанию: человеку ЛИБО компании (XOR).
// Уровень один — чтение: расписание ведёт его владелец.
type UserShare struct {
	ID         int64     `json:"id"`
	ScheduleID int64     `json:"schedule_id"`
	UserID     *int64    `json:"user_id"`
	CompanyID  *int64    `json:"company_id"`
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	// Name/AvatarPath/Login — подпись адресата в карточке «Поделиться».
	Name       string  `json:"name"`
	AvatarPath *string `json:"avatar_path"`
	Login      string  `json:"login"`
}

// User — идентичность пользователя для авторизации.
type User struct {
	ID           int64
	FIO          string
	AvatarPath   *string
	IsActive     bool
	IsSuperAdmin bool
}

// UserRef — коллега в списке «с кем поделиться».
type UserRef struct {
	ID         int64   `json:"id"`
	FIO        string  `json:"fio"`
	Login      string  `json:"login"`
	AvatarPath *string `json:"avatar_path"`
	Post       string  `json:"post"`
}

// CompanyRef — компания, которой можно открыть расписание.
type CompanyRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
