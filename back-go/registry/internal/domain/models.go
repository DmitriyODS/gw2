package domain

import (
	"time"

	"github.com/DmitriyODS/gw2/back-go/pkg/records"
)

// Уровни ролей в компании (общие с authsvc/tasksvc domain.Level*).
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelAdmin    = 3
)

// Типы полей реестра — общий набор pkg/records (продублирован во фронте,
// front/src/utils/registryFields.js — держать синхронным).
const (
	FieldImage    = records.FieldImage
	FieldFile     = records.FieldFile
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

// FieldTypes — допустимые типы (для валидации структуры реестра).
var FieldTypes = records.FieldTypes

// Registry — реестр: настраиваемая таблица-справочник. Принадлежит ЧЕЛОВЕКУ,
// коллеги и компании получают доступ шарингом.
type Registry struct {
	ID      int64 `json:"id"`
	OwnerID int64 `json:"owner_id"`
	// CompanyID — компания, в которой реестр заведён (nil — личный, вне
	// компании). Определяет, чья квота платит за файлы записей.
	CompanyID *int64 `json:"company_id"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	// SectionFieldID — поле-источник подразделов: его варианты становятся
	// вкладками над таблицей и фильтруют записи. Только поле типа select и
	// только своего реестра; nil — подразделы выключены.
	SectionFieldID *int64 `json:"section_field_id"`
	// Accounting — «Учётный реестр»: у записей появляется выдача с историей.
	Accounting bool      `json:"accounting"`
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// Fields — заполняется при чтении одного реестра / списка с полями.
	// Без omitempty: реестр без полей должен отдавать [] (а не отсутствующий
	// ключ), иначе на клиенте reg.fields === undefined.
	Fields []Field `json:"fields"`
	// MyAccess — эффективный уровень спрашивающего (см. access.go). Считает
	// сервер: клиенту нельзя доверять решение, что ему показывать.
	MyAccess string `json:"my_access"`
	// OwnerName — чей это реестр (вкладки «Поделились» и «Компания» должны
	// называть хозяина).
	OwnerName string `json:"owner_name,omitempty"`
}

// Field — поле (колонка карточки) реестра. Config хранит настройки конкретного
// типа: number → {pattern, min, max}; regex → {pattern, hint};
// select → {options, multiple}; datetime → {year, month, day, hours, minutes,
// seconds}; checkbox → {on_label, off_label}; phone → {country}.
type Field struct {
	ID          int64          `json:"id"`
	RegistryID  int64          `json:"registry_id"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Position    int            `json:"position"`
	ColSpan     int            `json:"col_span"` // 1..GridCols — ширина в сетке карточки
	RowSpan     int            `json:"row_span"` // ≥1 — высота
	ShowInTable bool           `json:"show_in_table"`
	CreatedAt   time.Time      `json:"created_at"`
}

// Record — запись реестра. Data — карта строкового field_id → значение
// (тип значения зависит от типа поля). SearchText не сериализуется наружу.
type Record struct {
	ID         int64          `json:"id"`
	RegistryID int64          `json:"registry_id"`
	Data       map[string]any `json:"data"`
	CreatedBy  *int64         `json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	// Issue — открытая выдача учётного реестра (nil — позиция на месте).
	// Приезжает батчем вместе со списком, без N+1.
	Issue *Issue `json:"issue,omitempty"`
}

// RecordScope — запись вместе с реестром, которому она принадлежит: разделу
// «Настройки → Хранилище» нужно показать, где лежит файл.
type RecordScope struct {
	Record       *Record
	RegistryID   int64
	RegistryName string
	CompanyID    int64
}

// ColumnFilter — условие по одному полю: тип поля решает, как сравнивать.
// Пустое значение — условия нет.
type ColumnFilter struct {
	FieldID int64
	// Op — вид сравнения: contains | equals | empty | filled | gt | lt | between | any.
	Op string
	// Values — операнды: одно значение у большинства, два у between, список у any.
	Values []string
}

// RecordListFilter — фильтры списка записей: поиск (по search_text), условия по
// колонкам, подраздел, сортировка по полю или дате создания, пагинация.
type RecordListFilter struct {
	RegistryID int64
	Search     string
	// SortFieldID — id поля для ORDER BY data->>'<id>'. 0 — сортировка по created_at.
	SortFieldID int64
	// SortKind — приведение типа при сортировке по полю: "number" | "date" | "text".
	SortKind string
	// SectionFieldID/SectionValue — вкладка-подраздел: значение поля-источника.
	// Поле спискового типа бывает и множественным, поэтому значение ищется и
	// как скаляр, и как элемент массива.
	SectionFieldID int64
	SectionValue   string
	// Columns — условия по отдельным полям (фильтры колонок таблицы).
	Columns []ColumnFilter
	Desc    bool
	Page    int
	PerPage int
}

// ExportFilter — набор записей под массовую операцию (выгрузка, удаление,
// печать QR). Либо явно перечисленные IDs, либо ВСЁ по фильтру экрана за
// вычетом Exclude: выбор «отметить всё» переживает страницы, и гонять на
// клиент тысячи id ради него не нужно — их множество описывается фильтром.
type ExportFilter struct {
	RegistryID     int64
	Search         string
	IDs            []int64
	Exclude        []int64
	SectionFieldID int64
	SectionValue   string
	Columns        []ColumnFilter
}

// Share — публичная ссылка на реестр. Code в URL — capability.
type Share struct {
	ID         int64  `json:"id"`
	RegistryID int64  `json:"registry_id"`
	Code       string `json:"code"`
	// Name — своё название ссылки: их у реестра много, и отзывать нужную
	// удобнее по слову, а не по коду.
	Name   string `json:"name"`
	Access string `json:"access"` // view | edit | admin
	// RequireAuth — открыть реестр можно только войдя в аккаунт. Тогда в
	// журнале посещений у перехода есть имя, а не только адрес.
	RequireAuth bool      `json:"require_auth"`
	CreatedBy   *int64    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
	// Visits/LastVisitAt — сводка журнала переходов (для карточки ссылки).
	Visits      int        `json:"visits"`
	LastVisitAt *time.Time `json:"last_visit_at,omitempty"`
}

// UserShare — адресный доступ: конкретному человеку либо всей компании.
type UserShare struct {
	ID         int64  `json:"id"`
	RegistryID int64  `json:"registry_id"`
	UserID     *int64 `json:"user_id"`
	CompanyID  *int64 `json:"company_id"`
	Access     string `json:"access"`
	// CreatedBy — КТО выдал доступ (не тот, кому выдали).
	CreatedBy *int64 `json:"created_by,omitempty"`
	// Name/AvatarPath — кому выдано (имя человека или название компании).
	Name       string    `json:"name"`
	AvatarPath *string   `json:"avatar_path,omitempty"`
	Login      string    `json:"login,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// ShareVisit — переход по внешней ссылке. У гостя аккаунта нет — остаётся
// только адрес и время.
type ShareVisit struct {
	ID        int64     `json:"id"`
	ShareID   int64     `json:"share_id"`
	UserID    *int64    `json:"user_id"`
	UserName  string    `json:"user_name,omitempty"`
	UserLogin string    `json:"user_login,omitempty"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	VisitedAt time.Time `json:"visited_at"`
}

// Issue — выдача позиции учётного реестра. Открытая (ReturnedAt == nil) у
// записи одна: её состояние показывает плашка в таблице.
type Issue struct {
	ID         int64 `json:"id"`
	RegistryID int64 `json:"registry_id"`
	RecordID   int64 `json:"record_id"`
	// IssuedTo — КОМУ ушла вещь: отдел, объект, бригада. Отвечает за неё
	// конкретный человек (HolderName), и это разные сведения.
	IssuedTo     string     `json:"issued_to"`
	HolderName   string     `json:"holder_name"`
	HolderPhone  string     `json:"holder_phone"`
	HolderUserID *int64     `json:"holder_user_id,omitempty"`
	IssuedBy     *int64     `json:"issued_by,omitempty"`
	IssuedByName string     `json:"issued_by_name,omitempty"`
	IssuedAt     time.Time  `json:"issued_at"`
	DueAt        *time.Time `json:"due_at,omitempty"`
	ReturnedAt   *time.Time `json:"returned_at,omitempty"`
	// Events — история движения (заполняется при чтении карточки выдачи).
	Events []IssueEvent `json:"events,omitempty"`
}

// Состояние позиции учётного реестра — производное от открытой выдачи.
const (
	StockIn      = "in"      // на месте
	StockIssued  = "issued"  // выдана, срок не вышел
	StockOverdue = "overdue" // срок прошёл
	StockNoDue   = "no_due"  // выдана без срока возврата
)

// State — состояние позиции на момент now. Считает СЕРВЕР: часовой пояс
// клиента и его часы не должны решать, просрочена ли вещь.
func (i *Issue) State(now time.Time) string {
	switch {
	case i == nil || i.ReturnedAt != nil:
		return StockIn
	case i.DueAt == nil:
		return StockNoDue
	case now.After(*i.DueAt):
		return StockOverdue
	default:
		return StockIssued
	}
}

// OverdueDays — на сколько суток просрочен возврат (0, если срок не вышел).
func (i *Issue) OverdueDays(now time.Time) int {
	if i == nil || i.ReturnedAt != nil || i.DueAt == nil || !now.After(*i.DueAt) {
		return 0
	}
	return int(now.Sub(*i.DueAt).Hours()/24) + 1
}

// Виды событий истории выдачи.
const (
	IssueEventIssue  = "issue"
	IssueEventExtend = "extend"
	IssueEventReturn = "return"
)

// IssueEvent — движение позиции: выдача, продление срока, возврат.
type IssueEvent struct {
	ID        int64      `json:"id"`
	IssueID   int64      `json:"issue_id"`
	Kind      string     `json:"kind"`
	DueAt     *time.Time `json:"due_at,omitempty"`
	Comment   string     `json:"comment"`
	ActorID   *int64     `json:"actor_id,omitempty"`
	ActorName string     `json:"actor_name,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// UploadedFile — метаданные загруженного файла/картинки (хранится в Data поля).
type UploadedFile struct {
	Path string `json:"path"` // относительный путь в uploads (раздаёт nginx /uploads/)
	Name string `json:"name"` // оригинальное имя файла
	Mime string `json:"mime"`
	Size int64  `json:"size"`
	// Thumb — уменьшенная копия картинки для таблицы записей; пусто, если
	// оригинал и так мал либо формат не декодируется (тогда показывается он сам).
	Thumb string `json:"thumb,omitempty"`
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

// SearchHit — строка глобального поиска (Hola): запись вместе с реестром,
// которому она принадлежит. Snippet — начало search_text записи.
type SearchHit struct {
	RegistryID   int64  `json:"registry_id"`
	RegistryName string `json:"registry_name"`
	RecordID     int64  `json:"record_id"`
	Snippet      string `json:"snippet"`
}
