package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/pkg/companydata"
)

/* Перенос компании: выгрузка и вливание раздела «Задачи».

   Формат — свой JSON, а не сырые строки таблиц: у переносимой компании
   меняются все идентификаторы, поэтому связи (задача → отдел, юнит → задача)
   восстанавливаются по картам старый id → новый. Люди в архив не входят —
   ссылки на них приходят уже сопоставленными (companydata.Import.UserID).

   Не переносится: эмбеддинги (пересчитываются), личные цвета и отметки
   прочтения комментариев (личное, а не компанийное), привязки YouGile
   (указывают на чужое пространство). */

type companyDump struct {
	Departments []dumpNamed   `json:"departments"`
	UnitTypes   []dumpNamed   `json:"unit_types"`
	Stages      []dumpColored `json:"stages"`
	Tags        []dumpColored `json:"tags"`
	Tasks       []dumpTask    `json:"tasks"`
	Comments    []dumpComment `json:"comments"`
	Units       []dumpUnit    `json:"units"`
}

type dumpNamed struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type dumpColored struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Color *string `json:"color,omitempty"`
}

type dumpTask struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	CreatedAt    time.Time  `json:"created_at"`
	AuthorID     int64      `json:"author_id"`
	Responsible  *int64     `json:"responsible_user_id,omitempty"`
	ReceivedAt   time.Time  `json:"received_at"`
	DepartmentID int64      `json:"department_id"`
	StageID      *int64     `json:"stage_id,omitempty"`
	Deadline     *time.Time `json:"deadline,omitempty"`
	IsArchived   bool       `json:"is_archived"`
	ArchivedAt   *time.Time `json:"archived_at,omitempty"`
	TagIDs       []int64    `json:"tag_ids,omitempty"`
}

type dumpComment struct {
	TaskID    int64      `json:"task_id"`
	AuthorID  int64      `json:"author_id"`
	Text      string     `json:"text"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`
}

type dumpUnit struct {
	Name       string     `json:"name"`
	UserID     int64      `json:"user_id"`
	UnitTypeID int64      `json:"unit_type_id"`
	TaskID     int64      `json:"task_id"`
	IsEdited   bool       `json:"is_edited"`
	Start      time.Time  `json:"datetime_start"`
	End        *time.Time `json:"datetime_end,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ExportCompany — весь контент задач компании одним JSON.
func (r *Repo) ExportCompany(ctx context.Context, companyID int64) (companydata.Export, error) {
	var dump companyDump

	named := func(query string) ([]dumpNamed, error) {
		rows, err := r.pool.Query(ctx, query, companyID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []dumpNamed{}
		for rows.Next() {
			var it dumpNamed
			if err := rows.Scan(&it.ID, &it.Name); err != nil {
				return nil, err
			}
			out = append(out, it)
		}
		return out, rows.Err()
	}
	colored := func(query string) ([]dumpColored, error) {
		rows, err := r.pool.Query(ctx, query, companyID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		out := []dumpColored{}
		for rows.Next() {
			var it dumpColored
			if err := rows.Scan(&it.ID, &it.Name, &it.Color); err != nil {
				return nil, err
			}
			out = append(out, it)
		}
		return out, rows.Err()
	}

	var err error
	if dump.Departments, err = named(`SELECT id, name FROM departments WHERE company_id = $1 ORDER BY id`); err != nil {
		return companydata.Export{}, err
	}
	if dump.UnitTypes, err = named(`SELECT id, name FROM unit_types WHERE company_id = $1 ORDER BY id`); err != nil {
		return companydata.Export{}, err
	}
	if dump.Stages, err = colored(`SELECT id, name, color FROM stages WHERE company_id = $1 ORDER BY id`); err != nil {
		return companydata.Export{}, err
	}
	if dump.Tags, err = colored(`SELECT id, name, color FROM tags WHERE company_id = $1 ORDER BY id`); err != nil {
		return companydata.Export{}, err
	}

	// Метки задачи приезжают агрегатом — иначе на каждую задачу был бы запрос.
	rows, err := r.pool.Query(ctx, `
		SELECT t.id, t.name, t.created_at, t.author_id, t.responsible_user_id, t.received_at,
		       t.department_id, t.stage_id, t.deadline, t.is_archived, t.archived_at,
		       COALESCE(ARRAY_AGG(tt.tag_id) FILTER (WHERE tt.tag_id IS NOT NULL), '{}')
		  FROM tasks t
		  LEFT JOIN task_tags tt ON tt.task_id = t.id
		 WHERE t.company_id = $1
		 GROUP BY t.id
		 ORDER BY t.id`, companyID)
	if err != nil {
		return companydata.Export{}, err
	}
	defer rows.Close()
	dump.Tasks = []dumpTask{}
	for rows.Next() {
		var t dumpTask
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.AuthorID, &t.Responsible, &t.ReceivedAt,
			&t.DepartmentID, &t.StageID, &t.Deadline, &t.IsArchived, &t.ArchivedAt, &t.TagIDs); err != nil {
			return companydata.Export{}, err
		}
		dump.Tasks = append(dump.Tasks, t)
	}
	if err := rows.Err(); err != nil {
		return companydata.Export{}, err
	}

	if err := r.selectInto(ctx, `
		SELECT c.task_id, c.author_id, c.text, c.created_at, c.updated_at
		  FROM comments c JOIN tasks t ON t.id = c.task_id
		 WHERE t.company_id = $1 AND c.deleted_at IS NULL
		 ORDER BY c.id`, companyID, func(row pgx.Rows) error {
		var c dumpComment
		if err := row.Scan(&c.TaskID, &c.AuthorID, &c.Text, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return err
		}
		dump.Comments = append(dump.Comments, c)
		return nil
	}); err != nil {
		return companydata.Export{}, err
	}
	if dump.Comments == nil {
		dump.Comments = []dumpComment{}
	}

	if err := r.selectInto(ctx, `
		SELECT name, user_id, unit_type_id, task_id, is_edited, datetime_start, datetime_end, created_at
		  FROM units WHERE company_id = $1 ORDER BY id`, companyID, func(row pgx.Rows) error {
		var u dumpUnit
		if err := row.Scan(&u.Name, &u.UserID, &u.UnitTypeID, &u.TaskID, &u.IsEdited, &u.Start, &u.End, &u.CreatedAt); err != nil {
			return err
		}
		dump.Units = append(dump.Units, u)
		return nil
	}); err != nil {
		return companydata.Export{}, err
	}
	if dump.Units == nil {
		dump.Units = []dumpUnit{}
	}

	payload, err := json.Marshal(dump)
	if err != nil {
		return companydata.Export{}, err
	}
	return companydata.Export{Payload: payload, Count: len(dump.Tasks)}, nil
}

func (r *Repo) selectInto(ctx context.Context, query string, arg any, scan func(pgx.Rows) error) error {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

// ImportCompany — влить задачи в компанию, созданную под импорт. Всё одной
// транзакцией: половина перенесённой компании хуже, чем ничего.
func (r *Repo) ImportCompany(ctx context.Context, in companydata.Import) (int, error) {
	var dump companyDump
	if err := json.Unmarshal(in.Payload, &dump); err != nil {
		return 0, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	depts := map[int64]int64{}
	for _, d := range dump.Departments {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO departments (name, company_id) VALUES ($1, $2) RETURNING id`,
			d.Name, in.CompanyID).Scan(&id); err != nil {
			return 0, err
		}
		depts[d.ID] = id
	}

	unitTypes := map[int64]int64{}
	for _, t := range dump.UnitTypes {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO unit_types (name, company_id) VALUES ($1, $2) RETURNING id`,
			t.Name, in.CompanyID).Scan(&id); err != nil {
			return 0, err
		}
		unitTypes[t.ID] = id
	}

	stages := map[int64]int64{}
	for _, s := range dump.Stages {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO stages (company_id, name, color) VALUES ($1, $2, $3) RETURNING id`,
			in.CompanyID, s.Name, s.Color).Scan(&id); err != nil {
			return 0, err
		}
		stages[s.ID] = id
	}

	tags := map[int64]int64{}
	for _, t := range dump.Tags {
		var id int64
		if err := tx.QueryRow(ctx,
			`INSERT INTO tags (company_id, name, color) VALUES ($1, $2, $3) RETURNING id`,
			in.CompanyID, t.Name, t.Color).Scan(&id); err != nil {
			return 0, err
		}
		tags[t.ID] = id
	}

	tasks := map[int64]int64{}
	for _, t := range dump.Tasks {
		// Отдел у задачи обязателен: если исходный не доехал, задачу пропускаем —
		// битая ссылка сломала бы раздел целиком.
		dept, ok := depts[t.DepartmentID]
		if !ok {
			continue
		}
		var stage *int64
		if t.StageID != nil {
			if s, ok := stages[*t.StageID]; ok {
				stage = &s
			}
		}
		var responsible *int64
		if t.Responsible != nil {
			id := in.UserID(*t.Responsible)
			responsible = &id
		}
		var id int64
		if err := tx.QueryRow(ctx, `
			INSERT INTO tasks (name, created_at, author_id, responsible_user_id, received_at,
			                   department_id, stage_id, deadline, is_archived, archived_at, company_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11) RETURNING id`,
			t.Name, t.CreatedAt, in.UserID(t.AuthorID), responsible, t.ReceivedAt,
			dept, stage, t.Deadline, t.IsArchived, t.ArchivedAt, in.CompanyID).Scan(&id); err != nil {
			return 0, err
		}
		tasks[t.ID] = id

		for _, tagID := range t.TagIDs {
			newTag, ok := tags[tagID]
			if !ok {
				continue
			}
			if _, err := tx.Exec(ctx,
				`INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				id, newTag); err != nil {
				return 0, err
			}
		}
	}

	for _, c := range dump.Comments {
		taskID, ok := tasks[c.TaskID]
		if !ok {
			continue
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO comments (task_id, author_id, text, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`,
			taskID, in.UserID(c.AuthorID), c.Text, c.CreatedAt, c.UpdatedAt); err != nil {
			return 0, err
		}
	}

	for _, u := range dump.Units {
		taskID, okTask := tasks[u.TaskID]
		typeID, okType := unitTypes[u.UnitTypeID]
		if !okTask || !okType {
			continue
		}
		// Незакрытый юнит закрываем на месте: активный юнит у пользователя
		// может быть только один, а импорт не должен занимать его чужой работой.
		end := u.End
		if end == nil {
			end = &u.Start
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO units (name, user_id, unit_type_id, task_id, is_edited,
			                   datetime_start, datetime_end, created_at, company_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			u.Name, in.UserID(u.UserID), typeID, taskID, u.IsEdited,
			u.Start, end, u.CreatedAt, in.CompanyID); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(tasks), nil
}
