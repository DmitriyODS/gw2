package domain

import "time"

// Уровни ролей в компании (общие с authsvc/tasksvc domain.Level*).
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelAdmin    = 3
)

// Типы полей карточки записи календаря. Набор совпадает с реестрами и
// продублирован во фронте (front/src/utils/registryFields.js) — держать
// синхронным. Встроенное поле «Дата и время» хранится отдельной колонкой
// calendar_records.event_at и в этот набор НЕ входит.
const (
	FieldImage    = "image"    // картинка (превью + полноэкранный просмотр)
	FieldFile     = "file"     // произвольный файл
	FieldText     = "text"     // текстовое поле (config.multiline — textarea)
	FieldNumber   = "number"   // число (config.pattern — опц. regex шаблона)
	FieldCheckbox = "checkbox" // галочка
	FieldSelect   = "select"   // выбор из вариантов (config.options, config.multiple)
	FieldLink     = "link"     // ссылка на сайт
	FieldDatetime = "datetime" // дата/время (config.year/month_day/time — части)
)

// FieldTypes — допустимые типы (для валидации структуры календаря).
var FieldTypes = map[string]bool{
	FieldImage: true, FieldFile: true, FieldText: true, FieldNumber: true,
	FieldCheckbox: true, FieldSelect: true, FieldLink: true, FieldDatetime: true,
}

// Calendar — календарь компании: набор полей карточки + записи, привязанные
// к дате/времени (см. Entry.EventAt).
type Calendar struct {
	ID        int64     `json:"id"`
	CompanyID int64     `json:"company_id"`
	Name      string    `json:"name"`
	Position  int       `json:"position"`
	CreatedBy *int64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Fields — заполняется при чтении одного календаря / списка с полями.
	// Без omitempty: календарь без полей должен отдавать [] (а не отсутствующий
	// ключ), иначе на клиенте cal.fields === undefined.
	Fields []Field `json:"fields"`
}

// Field — поле (часть карточки) записи. Config хранит настройки конкретного
// типа: number → {"pattern": "..."}; select → {"options": [...], "multiple": bool};
// datetime → {"year": bool, "month_day": bool, "time": bool}; text → {"multiline": bool}.
//
// Условная видимость (VisibleFieldID/VisibleValue): поле показывается в карточке
// только когда значение поля VisibleFieldID совпадает с VisibleValue (для
// checkbox-источника VisibleValue == "true" — «когда галочка отмечена», для
// select — конкретный выбранный вариант). nil — поле видно всегда.
type Field struct {
	ID             int64          `json:"id"`
	CalendarID     int64          `json:"calendar_id"`
	Label          string         `json:"label"`
	Type           string         `json:"type"`
	Config         map[string]any `json:"config"`
	Position       int            `json:"position"`
	ColSpan        int            `json:"col_span"` // 1..3 — ширина в сетке карточки
	RowSpan        int            `json:"row_span"` // ≥1 — высота
	ShowInTable    bool           `json:"show_in_table"`
	ShowInCard     bool           `json:"show_in_card"` // показывать в карточке события (виды день/неделя)
	VisibleFieldID *int64         `json:"visible_field_id"`
	VisibleValue   *string        `json:"visible_value"`
	CreatedAt      time.Time      `json:"created_at"`
}

// Entry — запись календаря. EventAt — обязательная дата/время (без секунд),
// по ней запись попадает в конкретный день. Data — карта строкового field_id →
// значение (тип зависит от типа поля). SearchText не сериализуется наружу.
type Entry struct {
	ID         int64          `json:"id"`
	CalendarID int64          `json:"calendar_id"`
	EventAt    time.Time      `json:"event_at"`
	Data       map[string]any `json:"data"`
	CreatedBy  *int64         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// EntryListFilter — фильтры выборки записей: диапазон дат (для просмотра
// дня/недели/месяца), сквозной поиск по тексту полей.
type EntryListFilter struct {
	CalendarID int64
	Search     string
	From       *time.Time // включительно
	To         *time.Time // НЕ включительно (полуинтервал)
	Limit      int
}

// Share — публичная ссылка на календарь (read-only, без авторизации). Code в
// URL — capability.
type Share struct {
	ID         int64     `json:"id"`
	CalendarID int64     `json:"calendar_id"`
	Code       string    `json:"code"`
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// UploadedFile — метаданные загруженного файла/картинки (хранится в Data поля).
type UploadedFile struct {
	Path string `json:"path"` // относительный путь в uploads (раздаёт nginx /uploads/)
	Name string `json:"name"` // оригинальное имя файла
	Mime string `json:"mime"`
	Size int64  `json:"size"`
}

// User — идентичность пользователя для авторизации (компания/роль из токена).
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	RoleLevel     int
	CompanyID     *int64
	CompanyActive bool
}
