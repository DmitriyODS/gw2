package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

const responseCols = `id, form_id, user_id, email, name, answers, score, max_score,
	graded, share_id, created_at, updated_at`

func responseScanTargets(r *domain.Response) []any {
	return []any{
		&r.ID, &r.FormID, &r.UserID, &r.Email, &r.Name, &r.Answers, &r.Score,
		&r.MaxScore, &r.Graded, &r.ShareID, &r.CreatedAt, &r.UpdatedAt,
	}
}

// uniqueViolation — «один ответ от человека» отбила база (частичный уникальный
// индекс): гонку двух вкладок ловим здесь, а не проверкой заранее.
func uniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

/*
ListResponses — страница собранных ответов.

	ORDER BY складывается из БЕЛОГО СПИСКА: в текст запроса попадают только
	константы файла, всё остальное — плейсхолдеры.
*/
func (r *Repo) ListResponses(ctx context.Context, f domain.ResponseListFilter) ([]*domain.Response, int, error) {
	order := "fr.created_at"
	if f.Sort == "score" {
		order = "fr.score"
	}
	direction := "ASC"
	if f.Desc {
		direction = "DESC"
	}
	offset := (f.Page - 1) * f.PerPage

	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT count(*) FROM form_responses fr
		 WHERE fr.form_id = $1 AND ($2 = '' OR fr.search_text ILIKE '%' || $2 || '%')`,
		f.FormID, f.Search).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s, COALESCE(u.fio, ''), u.avatar_path
		  FROM form_responses fr
		  LEFT JOIN users u ON u.id = fr.user_id
		 WHERE fr.form_id = $1 AND ($2 = '' OR fr.search_text ILIKE '%%' || $2 || '%%')
		 ORDER BY %s %s, fr.id %s
		 LIMIT $3 OFFSET $4`, prefixed(responseCols, "fr"), order, direction, direction),
		f.FormID, f.Search, f.PerPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := []*domain.Response{}
	for rows.Next() {
		var resp domain.Response
		targets := append(responseScanTargets(&resp), &resp.UserName, &resp.UserAvatar)
		if err := rows.Scan(targets...); err != nil {
			return nil, 0, err
		}
		out = append(out, &resp)
	}
	return out, total, rows.Err()
}

func (r *Repo) GetResponse(ctx context.Context, id int64) (*domain.Response, error) {
	var resp domain.Response
	err := r.pool.QueryRow(ctx, `
		SELECT `+prefixed(responseCols, "fr")+`, COALESCE(u.fio, ''), u.avatar_path
		  FROM form_responses fr
		  LEFT JOIN users u ON u.id = fr.user_id
		 WHERE fr.id = $1`, id).
		Scan(append(responseScanTargets(&resp), &resp.UserName, &resp.UserAvatar)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ResponseOfUser — последний ответ человека на форму.
func (r *Repo) ResponseOfUser(ctx context.Context, formID, userID int64) (*domain.Response, error) {
	var resp domain.Response
	err := r.pool.QueryRow(ctx, `
		SELECT `+responseCols+`
		  FROM form_responses
		 WHERE form_id = $1 AND user_id = $2
		 ORDER BY created_at DESC
		 LIMIT 1`, formID, userID).Scan(responseScanTargets(&resp)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (r *Repo) CountResponses(ctx context.Context, formID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx,
		`SELECT count(*) FROM form_responses WHERE form_id = $1`, formID).Scan(&n)
	return n, err
}

/*
CreateResponse — записать ответ.

	Копию настройки «один ответ от человека» берём из самой формы тем же
	запросом: между проверкой в сервисе и вставкой настройку могли поменять, а
	сторожить единственность обязана база.

	Места вопросов «Запись» сверяются В ТОЙ ЖЕ транзакции под локом формы —
	иначе последнее место достанется обоим, кто нажал «Отправить» одновременно.
*/
func (r *Repo) CreateResponse(ctx context.Context, resp *domain.Response, searchText string, bookings []domain.Booking) error {
	return r.inBookingTx(ctx, resp.FormID, bookings, 0, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			INSERT INTO form_responses
			    (form_id, user_id, email, name, answers, search_text, score, max_score,
			     graded, share_id, single, ip, user_agent)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			        COALESCE((SELECT one_response FROM forms WHERE id = $1), FALSE), $11, $12)
			RETURNING id, created_at, updated_at`,
			resp.FormID, resp.UserID, resp.Email, resp.Name, resp.Answers, searchText,
			resp.Score, resp.MaxScore, resp.Graded, resp.ShareID, resp.IP, resp.UserAgent).
			Scan(&resp.ID, &resp.CreatedAt, &resp.UpdatedAt)
		if uniqueViolation(err) {
			return domain.ErrAlreadyAnswered
		}
		return err
	})
}

func (r *Repo) UpdateResponse(ctx context.Context, resp *domain.Response, searchText string, bookings []domain.Booking) error {
	return r.inBookingTx(ctx, resp.FormID, bookings, resp.ID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `
			UPDATE form_responses
			   SET email=$2, answers=$3, search_text=$4, score=$5, max_score=$6,
			       graded=$7, updated_at = now()
			 WHERE id=$1
			 RETURNING updated_at`,
			resp.ID, resp.Email, resp.Answers, searchText, resp.Score, resp.MaxScore,
			resp.Graded).Scan(&resp.UpdatedAt)
	})
}

/*
inBookingTx — сохранение ответа с проверкой мест.

	Лок берётся на саму форму (pg_advisory_xact_lock), поэтому параллельные
	отправки выстраиваются в очередь ровно там, где считается остаток, и живут
	до конца транзакции. Без броней лок не нужен — обычная запись идёт мимо.
*/
func (r *Repo) inBookingTx(ctx context.Context, formID int64, bookings []domain.Booking,
	exceptResponse int64, save func(pgx.Tx) error) error {

	if len(bookings) == 0 {
		return pgx.BeginFunc(ctx, r.pool, save)
	}
	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, formID); err != nil {
			return err
		}
		for _, b := range bookings {
			var taken int
			if err := tx.QueryRow(ctx, `
				SELECT count(*) FROM form_responses
				 WHERE form_id = $1 AND answers ->> $2 = $3 AND ($4 = 0 OR id <> $4)`,
				formID, b.QuestionKey, b.Option, exceptResponse).Scan(&taken); err != nil {
				return err
			}
			if taken >= b.Capacity {
				return domain.ErrNoSlots
			}
		}
		return save(tx)
	})
}

/*
BookingCounts — сколько мест уже занято по каждому варианту «Записи».

	Ключ вопроса приходит параметром (`answers ->> $n`), а не подставляется в
	текст: инвариант платформы — внешние данные в SQL только плейсхолдерами.
*/
func (r *Repo) BookingCounts(ctx context.Context, formID int64, questionKeys []string, exceptResponse int64) (map[string]map[string]int, error) {
	out := map[string]map[string]int{}
	for _, key := range questionKeys {
		rows, err := r.pool.Query(ctx, `
			SELECT answers ->> $2 AS option, count(*)
			  FROM form_responses
			 WHERE form_id = $1 AND answers ? $2 AND ($3 = 0 OR id <> $3)
			 GROUP BY 1`, formID, key, exceptResponse)
		if err != nil {
			return nil, err
		}
		counts := map[string]int{}
		for rows.Next() {
			var option *string
			var n int
			if err := rows.Scan(&option, &n); err != nil {
				rows.Close()
				return nil, err
			}
			if option != nil {
				counts[*option] = n
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		out[key] = counts
	}
	return out, nil
}

func (r *Repo) DeleteResponse(ctx context.Context, formID, id int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM form_responses WHERE id = $1 AND form_id = $2`, id, formID)
	return err
}

// DeleteResponses — массовое удаление: перечисленные ответы либо все сразу.
// Возвращает удалённые (значения нужны чистке файлов, id — событию) одним
// запросом, без предварительной выборки.
func (r *Repo) DeleteResponses(ctx context.Context, formID int64, ids []int64, all bool) ([]*domain.Response, error) {
	if ids == nil {
		ids = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		DELETE FROM form_responses
		 WHERE form_id = $1 AND ($2 OR id = ANY($3))
		 RETURNING `+responseCols, formID, all, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Response{}
	for rows.Next() {
		var resp domain.Response
		if err := rows.Scan(responseScanTargets(&resp)...); err != nil {
			return nil, err
		}
		out = append(out, &resp)
	}
	return out, rows.Err()
}

// EachResponse — обход всех ответов формы потоком (сводка, выгрузка, чистка).
func (r *Repo) EachResponse(ctx context.Context, formID int64, fn func(*domain.Response) error) error {
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(responseCols, "fr")+`, COALESCE(u.fio, ''), u.avatar_path
		  FROM form_responses fr
		  LEFT JOIN users u ON u.id = fr.user_id
		 WHERE fr.form_id = $1
		 ORDER BY fr.created_at`, formID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var resp domain.Response
		targets := append(responseScanTargets(&resp), &resp.UserName, &resp.UserAvatar)
		if err := rows.Scan(targets...); err != nil {
			return err
		}
		if err := fn(&resp); err != nil {
			return err
		}
	}
	return rows.Err()
}

// SetResponsesSingle — синхронизировать копию настройки «один ответ». Ошибка
// уникальности означает, что кто-то уже отправил больше одного ответа.
func (r *Repo) SetResponsesSingle(ctx context.Context, formID int64, single bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE form_responses SET single = $2 WHERE form_id = $1 AND single <> $2`,
		formID, single)
	return err
}

// PublishGrades — открыть оценки теста (responseID == 0 — все ответы формы).
func (r *Repo) PublishGrades(ctx context.Context, formID, responseID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE form_responses SET graded = TRUE, updated_at = now()
		 WHERE form_id = $1 AND ($2 = 0 OR id = $2) AND NOT graded`, formID, responseID)
	return err
}

// ResponsesOfOwner — ответы вместе с их формой: раздел «Настройки → Хранилище»
// показывает, к какой форме приложен файл. Скоуп — формы человека и формы
// компаний, чью квоту он оплачивает (их присылает биллинг).
func (r *Repo) ResponsesOfOwner(ctx context.Context, userID int64, companyIDs []int64) ([]*domain.ResponseScope, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(responseCols, "fr")+`,
		       f.id, f.title, COALESCE(f.company_id, 0), f.owner_id
		  FROM form_responses fr
		  JOIN forms f ON f.id = fr.form_id
		 WHERE f.owner_id = $1 OR f.company_id = ANY($2)`, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.ResponseScope{}
	for rows.Next() {
		var resp domain.Response
		scope := &domain.ResponseScope{Response: &resp}
		targets := append(responseScanTargets(&resp),
			&scope.FormID, &scope.FormTitle, &scope.CompanyID, &scope.OwnerID)
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		out = append(out, scope)
	}
	return out, rows.Err()
}
