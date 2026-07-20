// Package domain — модели и порты микросервиса задач.
//
// tasksvc владеет таблицами tasks, units, unit_types, stages, departments,
// comments, favorites, user_task_colors (в рантайме); схему ведёт
// migrate-контейнер (goose). users/roles/companies читаются read-only.
package domain

import "time"

// Уровни ролей в компании (общие с authsvc domain.Level*). Системной роли 4
// больше нет — платформенного администратора заменяет users.is_super_admin.
const (
	LevelEmployee = 1
	LevelManager  = 2
	LevelAdmin    = 3
)

// TaskColors — фиксированный набор цветов-тегов задач (синхронизирован с
// front/src/utils/taskColors.js и токенами --tag-* в tokens.css).
// Этапы используют тот же набор (STAGE_COLORS во Flask).
var TaskColors = []string{"red", "orange", "amber", "green", "teal", "blue", "violet", "pink"}

func ValidTaskColor(c string) bool {
	for _, v := range TaskColors {
		if v == c {
			return true
		}
	}
	return false
}

// UserRef — пользователь в объёме AuthorRefSchema (автор/ответственный/
// активный участник/контрибьютор/автор комментария).
type UserRef struct {
	ID         int64
	FIO        string
	AvatarPath *string
}

// DeptRef / StageRef / TagRef — вложенные ссылки задачи.
type DeptRef struct {
	ID   int64
	Name string
}

// TagRef — тег в дампе задачи.
type TagRef struct {
	ID    int64
	Name  string
	Color string
}

// Tag — тег компании (справочник; цвет — из палитры TaskColors).
type Tag struct {
	ID        int64
	CompanyID int64
	Name      string
	Color     string
}

type StageRef struct {
	ID    int64
	Name  string
	Color string
	Order int
}

// Task — задача с подгруженными ссылками (как _TASK_LOAD_OPTIONS во Flask).
type Task struct {
	ID                int64
	Name              string
	CreatedAt         time.Time
	AuthorID          int64
	Author            *UserRef
	ResponsibleUserID *int64
	Responsible       *UserRef
	LinkYougile       *string
	ReceivedAt        time.Time
	DepartmentID      int64
	Department        *DeptRef
	StageID           *int64
	Stage             *StageRef
	Deadline          *time.Time
	IsArchived        bool
	ArchivedAt        *time.Time
	CompanyID         int64

	YougileTaskID    *string
	YougileIDShort   *string
	YougileProjectID *string
	YougileBoardID   *string
	YougileColumnID  *string
	// YougileSyncHash — антицикл двусторонней синхры (в дампы не попадает).
	YougileSyncHash *string
}

// Unit — юнит с user/unit_type (грузятся батчем для сериализации).
type Unit struct {
	ID            int64
	Name          string
	UserID        int64
	User          *UserRef
	UnitTypeID    int64
	UnitType      *UnitTypeRef
	TaskID        int64
	CompanyID     int64
	IsEdited      bool
	DatetimeStart time.Time
	DatetimeEnd   *time.Time
	CreatedAt     time.Time
}

type UnitTypeRef struct {
	ID   int64
	Name string
}

type UnitType struct {
	ID        int64
	Name      string
	CompanyID int64
}

type Department struct {
	ID        int64
	Name      string
	CompanyID int64
}

type Stage struct {
	ID        int64
	CompanyID int64
	Name      string
	Color     string
	Order     int
}

type Comment struct {
	ID        int64
	TaskID    int64
	AuthorID  int64
	Author    *UserRef
	Text      string
	CreatedAt time.Time
	UpdatedAt *time.Time
	DeletedAt *time.Time
}

// User — пользователь в объёме auth-мидлвари и проверок task-домена.
// Идентичность (id, fio, avatar, is_active, is_super_admin) грузится из users;
// активная компания и роль в ней (CompanyID/RoleLevel) заполняются из токена в
// authSource, CompanyActive — активность выбранной компании.
type User struct {
	ID            int64
	FIO           string
	AvatarPath    *string
	IsActive      bool
	IsSuperAdmin  bool
	OnVacation    bool // режим «в отпуске»: создание/правка задач и старт юнитов закрыты
	RoleLevel     int
	CompanyID     *int64
	CompanyActive bool
}

// TaskListFilter — фильтры списка задач (как task_repo.get_list).
type TaskListFilter struct {
	CurrentUserID     int64
	CompanyID         *int64
	Tab               string // active | favorites | archive
	Search            string
	Sort              string // last_activity | created_at | received_at | deadline
	DeptID            *int64
	StageID           *int64
	ResponsibleUserID *int64
	ReceivedFrom      *time.Time
	ReceivedTo        *time.Time
	HasUnits          string // "" | none | mine
	AuthorID          *int64
	// TagIDs — задача должна иметь ХОТЯ БЫ ОДИН из выбранных тегов (ИЛИ).
	TagIDs []int64
	// Colors — ЛИЧНЫЙ цвет карточки текущего пользователя (user_task_colors):
	// хотя бы один из выбранных.
	Colors  []string
	Page    int
	PerPage int
	// OrderedIDs — семантическая выдача aisvc: id уже по релевантности.
	// nil — обычный режим; пустой непустой указатель — пустая выдача.
	OrderedIDs []int64
	OrderedSet bool
}

// TaskEnrichment — батч-обогащение списка задач (без N+1).
type TaskEnrichment struct {
	ActiveUsers map[int64][]UserRef
	UserColors  map[int64]string
	FavoriteIDs map[int64]bool
	WithUnits   map[int64]bool
}
