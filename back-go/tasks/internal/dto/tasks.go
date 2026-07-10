// Package dto — JSON-формы REST-ответов и сокет-событий, байт-в-байт
// совместимые с прежними marshmallow-схемами Flask (schemas/task.py,
// unit.py, stage.py, department.py, unit_type.py, comment.py).
// Порядок полей — алфавитный: jsonify во Flask сортировал ключи.
package dto

import (
	"fmt"
	"time"

	"github.com/DmitriyODS/gw2/back-go/tasks/internal/domain"
)

// JSONTime — формат marshmallow (ISO8601 с явным смещением +00:00).
type JSONTime time.Time

func (t JSONTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + time.Time(t).UTC().Format("2006-01-02T15:04:05.999999-07:00") + `"`), nil
}

func optTime(t *time.Time) *JSONTime {
	if t == nil {
		return nil
	}
	ts := JSONTime(*t)
	return &ts
}

// ISO — isoformat() для payload'ов событий (task:archived, unit:stopped).
func ISO(t time.Time) string {
	u := t.UTC()
	s := u.Format("2006-01-02T15:04:05")
	if us := u.Nanosecond() / 1000; us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	return s + "+00:00"
}

// UserRef — форма AuthorRefSchema / UserRefSchema / CommentAuthorSchema.
type UserRef struct {
	AvatarPath *string `json:"avatar_path"`
	FIO        string  `json:"fio"`
	ID         int64   `json:"id"`
}

func NewUserRef(u *domain.UserRef) *UserRef {
	if u == nil {
		return nil
	}
	return &UserRef{AvatarPath: u.AvatarPath, FIO: u.FIO, ID: u.ID}
}

func NewUserRefs(users []domain.UserRef) []UserRef {
	out := make([]UserRef, 0, len(users))
	for i := range users {
		out = append(out, *NewUserRef(&users[i]))
	}
	return out
}

type DeptRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type StageRef struct {
	Color string `json:"color"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Order int    `json:"order"`
}

// TagDTO — тег компании (справочник и вложенная ссылка задачи).
type TagDTO struct {
	Color string `json:"color"`
	ID    int64  `json:"id"`
	Name  string `json:"name"`
}

func NewTagRefs(tags []domain.TagRef) []TagDTO {
	out := make([]TagDTO, 0, len(tags))
	for _, t := range tags {
		out = append(out, TagDTO{Color: t.Color, ID: t.ID, Name: t.Name})
	}
	return out
}

func NewTag(t *domain.Tag) TagDTO {
	return TagDTO{Color: t.Color, ID: t.ID, Name: t.Name}
}

func NewTags(items []*domain.Tag) []TagDTO {
	out := make([]TagDTO, 0, len(items))
	for _, t := range items {
		out = append(out, NewTag(t))
	}
	return out
}

// Task — форма TaskSchema + поля _enrich_task (active_users, color,
// is_favorite, has_units).
type Task struct {
	ActiveUsers       []UserRef `json:"active_users"`
	ArchivedAt        *JSONTime `json:"archived_at"`
	Author            *UserRef  `json:"author"`
	AuthorID          int64     `json:"author_id"`
	Color             *string   `json:"color"`
	CompanyID         int64     `json:"company_id"`
	CreatedAt         JSONTime  `json:"created_at"`
	Deadline          *JSONTime `json:"deadline"`
	Department        *DeptRef  `json:"department"`
	DepartmentID      int64     `json:"department_id"`
	HasUnits          bool      `json:"has_units"`
	ID                int64     `json:"id"`
	IsArchived        bool      `json:"is_archived"`
	IsFavorite        bool      `json:"is_favorite"`
	LinkYougile       *string   `json:"link_yougile"`
	Name              string    `json:"name"`
	ReceivedAt        JSONTime  `json:"received_at"`
	Responsible       *UserRef  `json:"responsible"`
	ResponsibleUserID *int64    `json:"responsible_user_id"`
	Stage             *StageRef `json:"stage"`
	StageID           *int64    `json:"stage_id"`
	Tags              []TagDTO  `json:"tags"`
	YougileBoardID    *string   `json:"yougile_board_id"`
	YougileColumnID   *string   `json:"yougile_column_id"`
	YougileIDShort    *string   `json:"yougile_id_short"`
	YougileProjectID  *string   `json:"yougile_project_id"`
	YougileTaskID     *string   `json:"yougile_task_id"`
}

// TaskEnrich — обогащение дампа задачи (как аргументы _enrich_task).
type TaskEnrich struct {
	IsFavorite     bool
	HasUnits       bool
	ActiveUsers    []domain.UserRef
	Color          *string
	Tags           []domain.TagRef
	YougileEnabled bool
}

func NewTask(t *domain.Task, e TaskEnrich) Task {
	out := Task{
		ActiveUsers:       NewUserRefs(e.ActiveUsers),
		ArchivedAt:        optTime(t.ArchivedAt),
		Author:            NewUserRef(t.Author),
		AuthorID:          t.AuthorID,
		Color:             e.Color,
		CompanyID:         t.CompanyID,
		CreatedAt:         JSONTime(t.CreatedAt),
		Deadline:          optTime(t.Deadline),
		DepartmentID:      t.DepartmentID,
		HasUnits:          e.HasUnits,
		ID:                t.ID,
		IsArchived:        t.IsArchived,
		IsFavorite:        e.IsFavorite,
		LinkYougile:       t.LinkYougile,
		Name:              t.Name,
		ReceivedAt:        JSONTime(t.ReceivedAt),
		ResponsibleUserID: t.ResponsibleUserID,
		StageID:           t.StageID,
		Tags:              NewTagRefs(e.Tags),
		YougileBoardID:    t.YougileBoardID,
		YougileColumnID:   t.YougileColumnID,
		YougileIDShort:    t.YougileIDShort,
		YougileProjectID:  t.YougileProjectID,
		YougileTaskID:     t.YougileTaskID,
	}
	if t.Department != nil {
		out.Department = &DeptRef{ID: t.Department.ID, Name: t.Department.Name}
	}
	if t.Stage != nil {
		out.Stage = &StageRef{Color: t.Stage.Color, ID: t.Stage.ID,
			Name: t.Stage.Name, Order: t.Stage.Order}
	}
	if t.Responsible != nil {
		out.Responsible = NewUserRef(t.Responsible)
	}
	// YouGile-ссылку отдаём только если в компании включена эта интеграция.
	if !e.YougileEnabled {
		out.LinkYougile = nil
	}
	return out
}

// TaskBroadcast — payload сокет-события task:created/updated: тот же дамп
// БЕЗ поля color (цвет личный — получатели применят свой при fetch).
type TaskBroadcast struct {
	ActiveUsers       []UserRef `json:"active_users"`
	ArchivedAt        *JSONTime `json:"archived_at"`
	Author            *UserRef  `json:"author"`
	AuthorID          int64     `json:"author_id"`
	CompanyID         int64     `json:"company_id"`
	CreatedAt         JSONTime  `json:"created_at"`
	Deadline          *JSONTime `json:"deadline"`
	Department        *DeptRef  `json:"department"`
	DepartmentID      int64     `json:"department_id"`
	HasUnits          bool      `json:"has_units"`
	ID                int64     `json:"id"`
	IsArchived        bool      `json:"is_archived"`
	IsFavorite        bool      `json:"is_favorite"`
	LinkYougile       *string   `json:"link_yougile"`
	Name              string    `json:"name"`
	ReceivedAt        JSONTime  `json:"received_at"`
	Responsible       *UserRef  `json:"responsible"`
	ResponsibleUserID *int64    `json:"responsible_user_id"`
	Stage             *StageRef `json:"stage"`
	StageID           *int64    `json:"stage_id"`
	Tags              []TagDTO  `json:"tags"`
	YougileBoardID    *string   `json:"yougile_board_id"`
	YougileColumnID   *string   `json:"yougile_column_id"`
	YougileIDShort    *string   `json:"yougile_id_short"`
	YougileProjectID  *string   `json:"yougile_project_id"`
	YougileTaskID     *string   `json:"yougile_task_id"`
}

func NewTaskBroadcast(t Task) TaskBroadcast {
	return TaskBroadcast{
		ActiveUsers: t.ActiveUsers, ArchivedAt: t.ArchivedAt, Author: t.Author,
		AuthorID: t.AuthorID, CompanyID: t.CompanyID, CreatedAt: t.CreatedAt,
		Deadline: t.Deadline, Department: t.Department, DepartmentID: t.DepartmentID,
		HasUnits: t.HasUnits, ID: t.ID, IsArchived: t.IsArchived,
		IsFavorite: t.IsFavorite, LinkYougile: t.LinkYougile, Name: t.Name,
		ReceivedAt: t.ReceivedAt, Responsible: t.Responsible,
		ResponsibleUserID: t.ResponsibleUserID, Stage: t.Stage, StageID: t.StageID,
		// Теги общие для компании (не личные, в отличие от color) — уходят
		// в броадкаст как есть.
		Tags:           t.Tags,
		YougileBoardID: t.YougileBoardID, YougileColumnID: t.YougileColumnID,
		YougileIDShort: t.YougileIDShort, YougileProjectID: t.YougileProjectID,
		YougileTaskID: t.YougileTaskID,
	}
}

// TaskList — ответ GET /api/tasks.
type TaskList struct {
	Items   []Task `json:"items"`
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	Total   int    `json:"total"`
}

// Unit — форма UnitSchema.
type Unit struct {
	CreatedAt     JSONTime     `json:"created_at"`
	DatetimeEnd   *JSONTime    `json:"datetime_end"`
	DatetimeStart JSONTime     `json:"datetime_start"`
	ID            int64        `json:"id"`
	IsEdited      bool         `json:"is_edited"`
	Name          string       `json:"name"`
	TaskID        int64        `json:"task_id"`
	UnitType      *UnitTypeRef `json:"unit_type"`
	UnitTypeID    int64        `json:"unit_type_id"`
	User          *UserRef     `json:"user"`
	UserID        int64        `json:"user_id"`
}

type UnitTypeRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func NewUnit(u *domain.Unit) Unit {
	out := Unit{
		CreatedAt:     JSONTime(u.CreatedAt),
		DatetimeEnd:   optTime(u.DatetimeEnd),
		DatetimeStart: JSONTime(u.DatetimeStart),
		ID:            u.ID,
		IsEdited:      u.IsEdited,
		Name:          u.Name,
		TaskID:        u.TaskID,
		UnitTypeID:    u.UnitTypeID,
		UserID:        u.UserID,
	}
	if u.UnitType != nil {
		out.UnitType = &UnitTypeRef{ID: u.UnitType.ID, Name: u.UnitType.Name}
	}
	if u.User != nil {
		out.User = NewUserRef(u.User)
	}
	return out
}

func NewUnits(units []*domain.Unit) []Unit {
	out := make([]Unit, 0, len(units))
	for _, u := range units {
		out = append(out, NewUnit(u))
	}
	return out
}

// UnitType / Department / Stage — формы соответствующих схем.
type UnitTypeDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func NewUnitTypes(items []*domain.UnitType) []UnitTypeDTO {
	out := make([]UnitTypeDTO, 0, len(items))
	for _, ut := range items {
		out = append(out, UnitTypeDTO{ID: ut.ID, Name: ut.Name})
	}
	return out
}

type DepartmentDTO struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

func NewDepartments(items []*domain.Department) []DepartmentDTO {
	out := make([]DepartmentDTO, 0, len(items))
	for _, d := range items {
		out = append(out, DepartmentDTO{ID: d.ID, Name: d.Name})
	}
	return out
}

type StageDTO struct {
	Color     string `json:"color"`
	CompanyID int64  `json:"company_id"`
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Order     int    `json:"order"`
}

func NewStage(s *domain.Stage) StageDTO {
	return StageDTO{Color: s.Color, CompanyID: s.CompanyID, ID: s.ID,
		Name: s.Name, Order: s.Order}
}

func NewStages(items []*domain.Stage) []StageDTO {
	out := make([]StageDTO, 0, len(items))
	for _, s := range items {
		out = append(out, NewStage(s))
	}
	return out
}

// Comment — форма CommentSchema.
type Comment struct {
	Author    *UserRef  `json:"author"`
	AuthorID  int64     `json:"author_id"`
	CreatedAt JSONTime  `json:"created_at"`
	DeletedAt *JSONTime `json:"deleted_at"`
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Text      string    `json:"text"`
	UpdatedAt *JSONTime `json:"updated_at"`
}

func NewComment(c *domain.Comment) Comment {
	return Comment{
		Author:    NewUserRef(c.Author),
		AuthorID:  c.AuthorID,
		CreatedAt: JSONTime(c.CreatedAt),
		DeletedAt: optTime(c.DeletedAt),
		ID:        c.ID,
		TaskID:    c.TaskID,
		Text:      c.Text,
		UpdatedAt: optTime(c.UpdatedAt),
	}
}

func NewComments(items []*domain.Comment) []Comment {
	out := make([]Comment, 0, len(items))
	for _, c := range items {
		out = append(out, NewComment(c))
	}
	return out
}

// ── Запросы (после schema-валидации в транспорте) ────────────────

// TaskCreate — распарсенный POST /api/tasks (даты received_at/deadline —
// date-only, как fields.Date).
type TaskCreate struct {
	Name              string
	LinkYougile       *string
	DepartmentID      int64
	ReceivedAt        *time.Time
	Deadline          *time.Time
	ResponsibleUserID *int64
	StageID           *int64
}

// TaskUpdate — распарсенный PATCH /api/tasks/<id>: *Set = поле передано.
type TaskUpdate struct {
	Name              *string
	LinkYougile       *string
	LinkYougileSet    bool
	DepartmentID      *int64
	ReceivedAt        *time.Time
	ReceivedAtSet     bool
	Deadline          *time.Time
	DeadlineSet       bool
	ResponsibleUserID *int64
	ResponsibleSet    bool
	StageID           *int64
	StageSet          bool
}

// ChangedFields — имена переданных полей (для YouGile-пуша и реиндекса).
func (u TaskUpdate) ChangedFields() []string {
	var out []string
	if u.Name != nil {
		out = append(out, "name")
	}
	if u.LinkYougileSet {
		out = append(out, "link_yougile")
	}
	if u.DepartmentID != nil {
		out = append(out, "department_id")
	}
	if u.ReceivedAtSet {
		out = append(out, "received_at")
	}
	if u.DeadlineSet {
		out = append(out, "deadline")
	}
	if u.ResponsibleSet {
		out = append(out, "responsible_user_id")
	}
	if u.StageSet {
		out = append(out, "stage_id")
	}
	return out
}

// UnitUpdate — распарсенный PATCH /api/units/<id>.
type UnitUpdate struct {
	Name          *string
	UnitTypeID    *int64
	DatetimeStart *time.Time
	DatetimeEnd   *time.Time
	DatetimeEndSet bool
}
