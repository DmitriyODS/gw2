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
	FieldNumber   = records.FieldNumber
	FieldCheckbox = records.FieldCheckbox
	FieldSelect   = records.FieldSelect
	FieldLink     = records.FieldLink
	FieldDatetime = records.FieldDatetime
	FieldStock    = records.FieldStock
)

// FieldTypes — допустимые типы (для валидации структуры реестра).
var FieldTypes = records.FieldTypes

// Registry — реестр компании (таблица-справочник).
type Registry struct {
	ID        int64  `json:"id"`
	CompanyID int64  `json:"company_id"`
	Name      string `json:"name"`
	Position  int    `json:"position"`
	// TagFieldID — поле-источник тегов: его варианты показываются чипами над
	// таблицей и фильтруют записи. Только поле типа select и только своего
	// реестра; nil — теги выключены.
	TagFieldID *int64    `json:"tag_field_id"`
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	// Fields — заполняется при чтении одного реестра / списка с полями.
	// Без omitempty: реестр без полей должен отдавать [] (а не отсутствующий
	// ключ), иначе на клиенте reg.fields === undefined.
	Fields []Field `json:"fields"`
}

// Field — поле (колонка карточки) реестра. Config хранит настройки конкретного
// типа: number → {"pattern": "..."}; select → {"options": [...], "multiple": bool};
// datetime → {"year": bool, "month_day": bool, "time": bool}; text → {"multiline": bool}.
type Field struct {
	ID          int64          `json:"id"`
	RegistryID  int64          `json:"registry_id"`
	Label       string         `json:"label"`
	Type        string         `json:"type"`
	Config      map[string]any `json:"config"`
	Position    int            `json:"position"`
	ColSpan     int            `json:"col_span"` // 1..3 — ширина в сетке карточки
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
}

// RecordScope — запись вместе с реестром, которому она принадлежит: разделу
// «Настройки → Хранилище» нужно показать, где лежит файл.
type RecordScope struct {
	Record       *Record
	RegistryID   int64
	RegistryName string
	CompanyID    int64
}

// RecordListFilter — фильтры списка записей: поиск (по search_text), сортировка
// по полю или дате создания, пагинация.
type RecordListFilter struct {
	RegistryID int64
	Search     string
	// SortFieldID — id поля для ORDER BY data->>'<id>'. 0 — сортировка по created_at.
	SortFieldID int64
	// SortKind — приведение типа при сортировке по полю: "number" | "date" | "text".
	SortKind string
	// TagFieldID/TagValue — фильтр чипом-тегом: значение поля-источника тегов.
	// Поле спискового типа бывает и множественным, поэтому значение ищется и
	// как скаляр, и как элемент массива.
	TagFieldID int64
	TagValue   string
	Desc       bool
	Page       int
	PerPage    int
}

// ExportFilter — набор записей под массовую операцию (выгрузка, удаление).
// Либо явно перечисленные IDs, либо ВСЁ по фильтру экрана (поиск и чип-тег)
// за вычетом Exclude: выбор «отметить всё» переживает страницы, и гонять на
// клиент тысячи id ради него не нужно — их множество описывается фильтром.
type ExportFilter struct {
	RegistryID int64
	Search     string
	IDs        []int64
	Exclude    []int64
	TagFieldID int64
	TagValue   string
}

// Уровни доступа внешней ссылки. Свойство САМОЙ ссылки: владелец раздаёт
// разным людям разные коды и отзывает их по отдельности.
const (
	ShareView = "view" // только просмотр (и выгрузка)
	ShareEdit = "edit" // + добавление и правка записей
)

// NormalizeShareAccess — привести уровень к допустимому; незнакомое значение
// трактуем как просмотр: ошибиться в сторону меньших прав безопаснее.
func NormalizeShareAccess(access string) string {
	if access == ShareEdit {
		return ShareEdit
	}
	return ShareView
}

// Share — публичная ссылка на реестр (без авторизации). Code в URL — capability.
type Share struct {
	ID         int64     `json:"id"`
	RegistryID int64     `json:"registry_id"`
	Code       string    `json:"code"`
	Access     string    `json:"access"` // view | edit
	CreatedBy  *int64    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// UploadedFile — метаданные загруженного файла/картинки (хранится в Data поля).
type UploadedFile struct {
	Path string `json:"path"` // относительный путь в uploads (raздаёт nginx /uploads/)
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

// SearchHit — строка глобального поиска (Spotlight): запись вместе с реестром,
// которому она принадлежит. Snippet — начало search_text записи.
type SearchHit struct {
	RegistryID   int64  `json:"registry_id"`
	RegistryName string `json:"registry_name"`
	RecordID     int64  `json:"record_id"`
	Snippet      string `json:"snippet"`
}
