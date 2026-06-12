package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/auth/internal/domain"
)

// BackupStore — выгрузка/восстановление основных таблиц (порт
// back/app/services/backup_service.py: тот же состав полей и порядок
// операций; разделы, добавленные позже multi-tenant'а, бэкап не покрывает).
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
		Roles:       []domain.BackupRole{},
		Users:       []domain.BackupUser{},
		Departments: []domain.BackupDepartment{},
		Tasks:       []domain.BackupTask{},
		Favorites:   []domain.BackupFavorite{},
		UnitTypes:   []domain.BackupUnitType{},
		Units:       []domain.BackupUnit{},
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
		SELECT id, fio, login, hash_password, post, role_id, avatar_path,
		       is_default_pass, is_hidden, created_at
		  FROM users`,
		func(rows pgx.Rows) error {
			var u domain.BackupUser
			var createdAt *time.Time
			if err := rows.Scan(&u.ID, &u.FIO, &u.Login, &u.HashPassword, &u.Post,
				&u.RoleID, &u.AvatarPath, &u.IsDefaultPass, &u.IsHidden, &createdAt); err != nil {
				return err
			}
			u.CreatedAt = pyISO(createdAt)
			data.Users = append(data.Users, u)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `SELECT id, name FROM departments`,
		func(rows pgx.Rows) error {
			var d domain.BackupDepartment
			if err := rows.Scan(&d.ID, &d.Name); err != nil {
				return err
			}
			data.Departments = append(data.Departments, d)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT id, name, author_id, link_yougile, received_at, department_id,
		       deadline, is_archived, archived_at, created_at
		  FROM tasks`,
		func(rows pgx.Rows) error {
			var t domain.BackupTask
			var receivedAt, deadline, archivedAt, createdAt *time.Time
			if err := rows.Scan(&t.ID, &t.Name, &t.AuthorID, &t.LinkYougile, &receivedAt,
				&t.DepartmentID, &deadline, &t.IsArchived, &archivedAt, &createdAt); err != nil {
				return err
			}
			t.ReceivedAt, t.Deadline = pyISO(receivedAt), pyISO(deadline)
			t.ArchivedAt, t.CreatedAt = pyISO(archivedAt), pyISO(createdAt)
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

	if err := queryEach(ctx, b.pool, `SELECT id, name FROM unit_types`,
		func(rows pgx.Rows) error {
			var ut domain.BackupUnitType
			if err := rows.Scan(&ut.ID, &ut.Name); err != nil {
				return err
			}
			data.UnitTypes = append(data.UnitTypes, ut)
			return nil
		}); err != nil {
		return nil, err
	}

	if err := queryEach(ctx, b.pool, `
		SELECT id, name, user_id, unit_type_id, task_id, is_edited,
		       datetime_start, datetime_end, created_at
		  FROM units`,
		func(rows pgx.Rows) error {
			var u domain.BackupUnit
			var start, end, createdAt *time.Time
			if err := rows.Scan(&u.ID, &u.Name, &u.UserID, &u.UnitTypeID, &u.TaskID,
				&u.IsEdited, &start, &end, &createdAt); err != nil {
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
// CASCADE сносит и все ссылающиеся таблицы (как прежний Flask-импорт), затем
// вставки в исходном порядке и setval последовательностей. Всё в одной
// транзакции: любая ошибка откатывает целиком.
func (b *BackupStore) ImportData(ctx context.Context, data *domain.BackupData) error {
	tx, err := b.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx,
		`TRUNCATE units, favorites, tasks, unit_types, departments, users, roles RESTART IDENTITY CASCADE`); err != nil {
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
			INSERT INTO users (id, fio, login, hash_password, post, role_id, avatar_path,
			    is_default_pass, is_hidden, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			u.ID, u.FIO, u.Login, u.HashPassword, u.Post, u.RoleID, u.AvatarPath,
			u.IsDefaultPass, u.IsHidden, u.CreatedAt); err != nil {
			return err
		}
	}
	for _, d := range data.Departments {
		if _, err := tx.Exec(ctx,
			`INSERT INTO departments (id, name) VALUES ($1, $2)`,
			d.ID, d.Name); err != nil {
			return err
		}
	}
	for _, t := range data.Tasks {
		if _, err := tx.Exec(ctx, `
			INSERT INTO tasks (id, name, author_id, link_yougile, received_at, department_id,
			    deadline, is_archived, archived_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			t.ID, t.Name, t.AuthorID, t.LinkYougile, t.ReceivedAt, t.DepartmentID,
			t.Deadline, t.IsArchived, t.ArchivedAt, t.CreatedAt); err != nil {
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
	for _, ut := range data.UnitTypes {
		if _, err := tx.Exec(ctx,
			`INSERT INTO unit_types (id, name) VALUES ($1, $2)`,
			ut.ID, ut.Name); err != nil {
			return err
		}
	}
	for _, u := range data.Units {
		if _, err := tx.Exec(ctx, `
			INSERT INTO units (id, name, user_id, unit_type_id, task_id, is_edited,
			    datetime_start, datetime_end, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			u.ID, u.Name, u.UserID, u.UnitTypeID, u.TaskID, u.IsEdited,
			u.DatetimeStart, u.DatetimeEnd, u.CreatedAt); err != nil {
			return err
		}
	}

	for _, seq := range []string{"roles", "users", "departments", "tasks", "unit_types", "units"} {
		if _, err := tx.Exec(ctx,
			fmt.Sprintf("SELECT setval('%s_id_seq', (SELECT MAX(id) FROM %s))", seq, seq)); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
