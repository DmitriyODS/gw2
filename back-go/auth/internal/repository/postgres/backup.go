package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// BackupStore — выгрузка/восстановление идентичности (роли, пользователи,
// компании, членства) и ядра учёта задач (отделы, этапы, типы юнитов, задачи,
// избранное, юниты). Разделы вне этого ядра (мессенджер, звонки, «Мой Groove»,
// эмбеддинги) бэкап не покрывает — деструктивный импорт их очищает каскадом.
type BackupStore struct {
	pool *pgxpool.Pool
}

func NewBackupStore(pool *pgxpool.Pool) *BackupStore {
	return &BackupStore{pool: pool}
}

var _ domain.BackupStore = (*BackupStore)(nil)

// pyISO — datetime.isoformat() для timestamptz: микросекунды ровно 6 цифр
// (опускаются, если нулевые), смещение +00:00.
func pyISO(t *time.Time) *string {
	if t == nil {
		return nil
	}
	u := t.UTC()
	s := u.Format("2006-01-02T15:04:05")
	if us := u.Nanosecond() / 1000; us != 0 {
		s += fmt.Sprintf(".%06d", us)
	}
	s += "+00:00"
	return &s
}

func (b *BackupStore) ExportData(ctx context.Context) (*domain.BackupData, error) {
	data := &domain.BackupData{
		Roles:         []domain.BackupRole{},
		Users:         []domain.BackupUser{},
		Companies:     []domain.BackupCompany{},
		UserCompanies: []domain.BackupMembership{},
		Departments:   []domain.BackupDepartment{},
		Stages:        []domain.BackupStage{},
		UnitTypes:     []domain.BackupUnitType{},
		Tasks:         []domain.BackupTask{},
		Favorites:     []domain.BackupFavorite{},
		Units:         []domain.BackupUnit{},
	}

	if err := queryEach(ctx, b.pool, `SELECT id, name, level FROM roles`,
		func(rows pgx.Rows) error {
			var r domain.BackupRole
			if err := rows.Scan(&r.ID, &r.Name, &r.Level); err != nil {
				return err
			}
			data.Roles = append(data.Roles, r)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT id, fio, login, hash_password, avatar_path,
		       is_default_pass, is_active, is_super_admin, created_at
		  FROM users`,
		func(rows pgx.Rows) error {
			var u domain.BackupUser
			var createdAt *time.Time
			if err := rows.Scan(&u.ID, &u.FIO, &u.Login, &u.HashPassword, &u.AvatarPath,
				&u.IsDefaultPass, &u.IsActive, &u.IsSuperAdmin, &createdAt); err != nil {
				return err
			}
			u.CreatedAt = pyISO(createdAt)
			data.Users = append(data.Users, u)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT id, name, description, is_active, settings, created_at,
		       ai_enabled, ai_api_key_enc, ai_key_hint, ai_model_chat, ai_model_embedding,
		       yg_company_id, yg_company_name, yg_project_id, yg_project_title,
		       yg_board_id, yg_board_title, yg_first_column_id, yg_completed_column_id,
		       yg_webhook_id, yg_webhook_secret, invite_code, created_by
		  FROM companies`,
		func(rows pgx.Rows) error {
			var c domain.BackupCompany
			var settings []byte
			var createdAt *time.Time
			if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.IsActive, &settings, &createdAt,
				&c.AIEnabled, &c.AIAPIKeyEnc, &c.AIKeyHint, &c.AIModelChat, &c.AIModelEmbedding,
				&c.YgCompanyID, &c.YgCompanyName, &c.YgProjectID, &c.YgProjectTitle,
				&c.YgBoardID, &c.YgBoardTitle, &c.YgFirstColumnID, &c.YgCompletedColumnID,
				&c.YgWebhookID, &c.YgWebhookSecret, &c.InviteCode, &c.CreatedBy); err != nil {
				return err
			}
			c.Settings = json.RawMessage(settings)
			c.CreatedAt = pyISO(createdAt)
			data.Companies = append(data.Companies, c)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT user_id, company_id, role_id, post, created_at FROM user_companies`,
		func(rows pgx.Rows) error {
			var m domain.BackupMembership
			var createdAt *time.Time
			if err := rows.Scan(&m.UserID, &m.CompanyID, &m.RoleID, &m.Post, &createdAt); err != nil {
				return err
			}
			m.CreatedAt = pyISO(createdAt)
			data.UserCompanies = append(data.UserCompanies, m)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `SELECT id, name, company_id FROM departments`,
		func(rows pgx.Rows) error {
			var d domain.BackupDepartment
			if err := rows.Scan(&d.ID, &d.Name, &d.CompanyID); err != nil {
				return err
			}
			data.Departments = append(data.Departments, d)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `SELECT id, company_id, name, color, "order" FROM stages`,
		func(rows pgx.Rows) error {
			var s domain.BackupStage
			if err := rows.Scan(&s.ID, &s.CompanyID, &s.Name, &s.Color, &s.Order); err != nil {
				return err
			}
			data.Stages = append(data.Stages, s)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `SELECT id, name, company_id FROM unit_types`,
		func(rows pgx.Rows) error {
			var ut domain.BackupUnitType
			if err := rows.Scan(&ut.ID, &ut.Name, &ut.CompanyID); err != nil {
				return err
			}
			data.UnitTypes = append(data.UnitTypes, ut)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT id, name, author_id, link_yougile, received_at, department_id,
		       deadline, is_archived, archived_at, created_at, color, company_id,
		       responsible_user_id, stage_id, yougile_task_id, yougile_project_id,
		       yougile_board_id, yougile_column_id, yougile_synced_at, yougile_sync_hash,
		       yougile_id_short
		  FROM tasks`,
		func(rows pgx.Rows) error {
			var t domain.BackupTask
			var receivedAt, deadline, archivedAt, createdAt, yougileSyncedAt *time.Time
			if err := rows.Scan(&t.ID, &t.Name, &t.AuthorID, &t.LinkYougile, &receivedAt,
				&t.DepartmentID, &deadline, &t.IsArchived, &archivedAt, &createdAt, &t.Color,
				&t.CompanyID, &t.ResponsibleUserID, &t.StageID, &t.YougileTaskID, &t.YougileProjectID,
				&t.YougileBoardID, &t.YougileColumnID, &yougileSyncedAt, &t.YougileSyncHash,
				&t.YougileIDShort); err != nil {
				return err
			}
			t.ReceivedAt, t.Deadline = pyISO(receivedAt), pyISO(deadline)
			t.ArchivedAt, t.CreatedAt = pyISO(archivedAt), pyISO(createdAt)
			t.YougileSyncedAt = pyISO(yougileSyncedAt)
			data.Tasks = append(data.Tasks, t)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `SELECT task_id, user_id FROM favorites`,
		func(rows pgx.Rows) error {
			var f domain.BackupFavorite
			if err := rows.Scan(&f.TaskID, &f.UserID); err != nil {
				return err
			}
			data.Favorites = append(data.Favorites, f)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT id, name, user_id, unit_type_id, task_id, is_edited,
		       datetime_start, datetime_end, created_at, company_id
		  FROM units`,
		func(rows pgx.Rows) error {
			var u domain.BackupUnit
			var start, end, createdAt *time.Time
			if err := rows.Scan(&u.ID, &u.Name, &u.UserID, &u.UnitTypeID, &u.TaskID,
				&u.IsEdited, &start, &end, &createdAt, &u.CompanyID); err != nil {
				return err
			}
			u.DatetimeStart, u.DatetimeEnd, u.CreatedAt = pyISO(start), pyISO(end), pyISO(createdAt)
			data.Units = append(data.Units, u)
			return nil
		}); err != nil {
		return nil, err
	}

	return data, nil
}

func queryEach(ctx context.Context, pool *pgxpool.Pool, sql string, scan func(pgx.Rows) error) error {
	rows, err := pool.Query(ctx, sql)
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

// ImportData — ДЕСТРУКТИВНОЕ восстановление: TRUNCATE ... RESTART IDENTITY
// CASCADE сносит и все ссылающиеся таблицы (мессенджер, звонки, грувики и т.д.),
// затем вставки в FK-безопасном порядке и setval последовательностей. Всё в
// одной транзакции: любая ошибка откатывает целиком.
func (b *BackupStore) ImportData(ctx context.Context, data *domain.BackupData) error {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`TRUNCATE user_companies, units, favorites, tasks, stages, unit_types, departments, companies, users, roles RESTART IDENTITY CASCADE`); err != nil {
		return err
	}

	for _, r := range data.Roles {
		if _, err := tx.Exec(ctx,
			`INSERT INTO roles (id, name, level) VALUES ($1, $2, $3)`,
			r.ID, r.Name, r.Level); err != nil {
			return err
		}
	}
	for _, u := range data.Users {
		if _, err := tx.Exec(ctx, `
			INSERT INTO users (id, fio, login, hash_password, avatar_path,
			    is_default_pass, is_active, is_super_admin, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			u.ID, u.FIO, u.Login, u.HashPassword, u.AvatarPath,
			u.IsDefaultPass, u.IsActive, u.IsSuperAdmin, u.CreatedAt); err != nil {
			return err
		}
	}
	// Компании — после users (FK created_by → users).
	for _, c := range data.Companies {
		settings := c.Settings
		if len(settings) == 0 {
			settings = json.RawMessage("{}")
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO companies (id, name, description, is_active, settings, created_at,
			    ai_enabled, ai_api_key_enc, ai_key_hint, ai_model_chat, ai_model_embedding,
			    yg_company_id, yg_company_name, yg_project_id, yg_project_title,
			    yg_board_id, yg_board_title, yg_first_column_id, yg_completed_column_id,
			    yg_webhook_id, yg_webhook_secret, invite_code, created_by)
			VALUES ($1, $2, $3, $4, $5::jsonb, COALESCE($6::timestamptz, now()),
			    $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)`,
			c.ID, c.Name, c.Description, c.IsActive, string(settings), c.CreatedAt,
			c.AIEnabled, c.AIAPIKeyEnc, c.AIKeyHint, c.AIModelChat, c.AIModelEmbedding,
			c.YgCompanyID, c.YgCompanyName, c.YgProjectID, c.YgProjectTitle,
			c.YgBoardID, c.YgBoardTitle, c.YgFirstColumnID, c.YgCompletedColumnID,
			c.YgWebhookID, c.YgWebhookSecret, c.InviteCode, c.CreatedBy); err != nil {
			return err
		}
	}
	// Членства — после users и companies (FK на обе + roles).
	for _, m := range data.UserCompanies {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_companies (user_id, company_id, role_id, post, created_at)
			VALUES ($1, $2, $3, $4, COALESCE($5::timestamptz, now()))`,
			m.UserID, m.CompanyID, m.RoleID, m.Post, m.CreatedAt); err != nil {
			return err
		}
	}

	for _, d := range data.Departments {
		if _, err := tx.Exec(ctx,
			`INSERT INTO departments (id, name, company_id) VALUES ($1, $2, $3)`,
			d.ID, d.Name, d.CompanyID); err != nil {
			return err
		}
	}
	for _, s := range data.Stages {
		if _, err := tx.Exec(ctx,
			`INSERT INTO stages (id, company_id, name, color, "order") VALUES ($1, $2, $3, $4, $5)`,
			s.ID, s.CompanyID, s.Name, s.Color, s.Order); err != nil {
			return err
		}
	}
	for _, ut := range data.UnitTypes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO unit_types (id, name, company_id) VALUES ($1, $2, $3)`,
			ut.ID, ut.Name, ut.CompanyID); err != nil {
			return err
		}
	}
	// Задачи — после departments/stages/companies/users (все их FK).
	for _, t := range data.Tasks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tasks (id, name, author_id, link_yougile, received_at, department_id,
			    deadline, is_archived, archived_at, created_at, color, company_id,
			    responsible_user_id, stage_id, yougile_task_id, yougile_project_id,
			    yougile_board_id, yougile_column_id, yougile_synced_at, yougile_sync_hash,
			    yougile_id_short)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
			    $17, $18, $19, $20, $21)`,
			t.ID, t.Name, t.AuthorID, t.LinkYougile, t.ReceivedAt, t.DepartmentID,
			t.Deadline, t.IsArchived, t.ArchivedAt, t.CreatedAt, t.Color, t.CompanyID,
			t.ResponsibleUserID, t.StageID, t.YougileTaskID, t.YougileProjectID,
			t.YougileBoardID, t.YougileColumnID, t.YougileSyncedAt, t.YougileSyncHash,
			t.YougileIDShort); err != nil {
			return err
		}
	}
	for _, f := range data.Favorites {
		if _, err := tx.Exec(ctx,
			`INSERT INTO favorites (task_id, user_id) VALUES ($1, $2)`,
			f.TaskID, f.UserID); err != nil {
			return err
		}
	}
	for _, u := range data.Units {
		if _, err := tx.Exec(ctx, `
			INSERT INTO units (id, name, user_id, unit_type_id, task_id, is_edited,
			    datetime_start, datetime_end, created_at, company_id)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			u.ID, u.Name, u.UserID, u.UnitTypeID, u.TaskID, u.IsEdited,
			u.DatetimeStart, u.DatetimeEnd, u.CreatedAt, u.CompanyID); err != nil {
			return err
		}
	}

	// Последовательности — за макс. id (пустая таблица → 1, чтобы setval не
	// падал на NULL).
	for _, seq := range []string{"roles", "users", "companies", "departments", "stages", "tasks", "unit_types", "units"} {
		if _, err := tx.Exec(ctx, fmt.Sprintf(
			`SELECT setval('%s_id_seq', GREATEST(COALESCE((SELECT MAX(id) FROM %s), 1), 1))`,
			seq, seq)); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
