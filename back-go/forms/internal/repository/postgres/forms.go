package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

type Repo struct {
	pool *pgxpool.Pool
}

var _ domain.FormRepository = (*Repo)(nil)

func NewRepo(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const formCols = `id, owner_id, company_id, title, description, status,
	allow_anonymous, one_response, allow_edit, collect_email, collect_name, show_progress,
	shuffle_questions, confirmation, show_summary, quiz, quiz_release,
	quiz_show_answers, opens_at, closes_at, max_responses, position,
	created_by, created_at, updated_at`

// formScanTargets — поля формы в порядке formCols (список один, а читают его
// и одиночный запрос, и список с JOIN).
func formScanTargets(f *domain.Form) []any {
	return []any{
		&f.ID, &f.OwnerID, &f.CompanyID, &f.Title, &f.Description, &f.Status,
		&f.AllowAnonymous, &f.OneResponse, &f.AllowEdit, &f.CollectEmail, &f.CollectName,
		&f.ShowProgress,
		&f.ShuffleQuestions, &f.Confirmation, &f.ShowSummary, &f.Quiz, &f.QuizRelease,
		&f.QuizShowAnswers, &f.OpensAt, &f.ClosesAt, &f.MaxResponses, &f.Position,
		&f.CreatedBy, &f.CreatedAt, &f.UpdatedAt,
	}
}

/*
accessExpr — эффективный уровень доступа одним выражением.

	Уровень приходит человеку несколькими путями сразу (он владелец, ему выдали
	лично, выдали его компании), и брать нужно СИЛЬНЕЙШИЙ. Порядок уровней задан
	здесь числом, а не сравнением строк: 'view' > 'edit' лексикографически, и
	наивный MAX(access) молча понижал бы права. Держать в паре с domain/access.go.
*/
const accessExpr = `
	CASE WHEN f.owner_id = $1 THEN 'owner' ELSE COALESCE((
		SELECT CASE max(CASE sh.access
		            WHEN 'edit' THEN 3 WHEN 'view' THEN 2 ELSE 1 END)
		         WHEN 3 THEN 'edit' WHEN 2 THEN 'view' WHEN 1 THEN 'respond' END
		  FROM form_user_shares sh
		 WHERE sh.form_id = f.id
		   AND (sh.user_id = $1 OR sh.company_id = ANY($2))
	), '') END`

// scopeCondition — условие вкладки раздела.
func scopeCondition(scope string) string {
	switch scope {
	case domain.ScopeMine:
		return `f.owner_id = $1`
	case domain.ScopeAssigned:
		return `f.owner_id <> $1
		        AND EXISTS (SELECT 1 FROM form_user_shares sh
		                     WHERE sh.form_id = f.id AND sh.access = 'respond'
		                       AND (sh.user_id = $1 OR sh.company_id = ANY($2)))`
	case domain.ScopeShared:
		return `f.owner_id <> $1
		        AND EXISTS (SELECT 1 FROM form_user_shares sh
		                     WHERE sh.form_id = f.id AND sh.access IN ('view', 'edit')
		                       AND (sh.user_id = $1 OR sh.company_id = ANY($2)))`
	default:
		return `(f.owner_id = $1
		         OR EXISTS (SELECT 1 FROM form_user_shares sh
		                     WHERE sh.form_id = f.id
		                       AND (sh.user_id = $1 OR sh.company_id = ANY($2))))`
	}
}

// ListForms — формы области вместе с уровнем доступа, именем владельца, числом
// собранных ответов и собственной обязанностью спрашивающего: карточка списка
// показывает всё это сразу, поэтому и считается одним запросом.
func (r *Repo) ListForms(ctx context.Context, userID int64, companyIDs []int64, scope string) ([]*domain.Form, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+prefixed(formCols, "f")+`, `+accessExpr+`, COALESCE(u.fio, ''),
		       (SELECT count(*) FROM form_responses fr WHERE fr.form_id = f.id),
		       (SELECT min(sh.due_at) FROM form_user_shares sh
		         WHERE sh.form_id = f.id AND sh.access = 'respond'
		           AND (sh.user_id = $1 OR sh.company_id = ANY($2))),
		       EXISTS (SELECT 1 FROM form_responses fr
		                WHERE fr.form_id = f.id AND fr.user_id = $1)
		  FROM forms f
		  LEFT JOIN users u ON u.id = f.owner_id
		 WHERE `+scopeCondition(scope)+`
		 ORDER BY f.updated_at DESC, f.id DESC`, userID, companyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*domain.Form{}
	for rows.Next() {
		var f domain.Form
		targets := append(formScanTargets(&f),
			&f.MyAccess, &f.OwnerName, &f.Responses, &f.MyDueAt, &f.MyResponded)
		if err := rows.Scan(targets...); err != nil {
			return nil, err
		}
		f.Sections = []domain.Section{}
		out = append(out, &f)
	}
	return out, rows.Err()
}

func (r *Repo) GetForm(ctx context.Context, id int64) (*domain.Form, error) {
	var f domain.Form
	err := r.pool.QueryRow(ctx,
		`SELECT `+formCols+`,
		        (SELECT count(*) FROM form_responses fr WHERE fr.form_id = forms.id)
		   FROM forms WHERE id = $1`, id).
		Scan(append(formScanTargets(&f), &f.Responses)...)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (r *Repo) CountOwned(ctx context.Context, ownerID int64) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `SELECT count(*) FROM forms WHERE owner_id = $1`, ownerID).Scan(&n)
	return n, err
}

func (r *Repo) CreateForm(ctx context.Context, f *domain.Form) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO forms (owner_id, company_id, title, description, status,
		    allow_anonymous, one_response, allow_edit, collect_email, collect_name,
		    show_progress, shuffle_questions, confirmation, show_summary, quiz,
		    quiz_release, quiz_show_answers, opens_at, closes_at, max_responses,
		    position, created_by)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		RETURNING id, created_at, updated_at`,
		f.OwnerID, f.CompanyID, f.Title, f.Description, f.Status,
		f.AllowAnonymous, f.OneResponse, f.AllowEdit, f.CollectEmail, f.CollectName,
		f.ShowProgress,
		f.ShuffleQuestions, f.Confirmation, f.ShowSummary, f.Quiz, f.QuizRelease,
		f.QuizShowAnswers, f.OpensAt, f.ClosesAt, f.MaxResponses, f.Position, f.CreatedBy).
		Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

func (r *Repo) UpdateForm(ctx context.Context, f *domain.Form) error {
	return r.pool.QueryRow(ctx, `
		UPDATE forms
		   SET title=$2, description=$3, status=$4, allow_anonymous=$5, one_response=$6,
		       allow_edit=$7, collect_email=$8, collect_name=$9, show_progress=$10,
		       shuffle_questions=$11, confirmation=$12, show_summary=$13, quiz=$14,
		       quiz_release=$15, quiz_show_answers=$16, opens_at=$17, closes_at=$18,
		       max_responses=$19, updated_at = now()
		 WHERE id=$1
		 RETURNING updated_at`,
		f.ID, f.Title, f.Description, f.Status, f.AllowAnonymous, f.OneResponse,
		f.AllowEdit, f.CollectEmail, f.CollectName, f.ShowProgress, f.ShuffleQuestions,
		f.Confirmation, f.ShowSummary, f.Quiz, f.QuizRelease, f.QuizShowAnswers,
		f.OpensAt, f.ClosesAt, f.MaxResponses).Scan(&f.UpdatedAt)
}

func (r *Repo) DeleteForm(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM forms WHERE id = $1`, id)
	return err
}

func (r *Repo) NextPosition(ctx context.Context, ownerID int64) (int, error) {
	var pos int
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(position), 0) + 1 FROM forms WHERE owner_id = $1`, ownerID).Scan(&pos)
	return pos, err
}

// SearchForms — строка поиска Hola: доступные формы по названию и описанию.
func (r *Repo) SearchForms(ctx context.Context, userID int64, companyIDs []int64, query string, limit int) ([]*domain.SearchHit, error) {
	if companyIDs == nil {
		companyIDs = []int64{}
	}
	rows, err := r.pool.Query(ctx, `
		SELECT f.id, f.title, left(f.description, 160), f.status
		  FROM forms f
		 WHERE `+scopeCondition(domain.ScopeAll)+`
		   AND (f.title ILIKE '%' || $3 || '%' OR f.description ILIKE '%' || $3 || '%')
		 ORDER BY f.updated_at DESC
		 LIMIT $4`, userID, companyIDs, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.SearchHit{}
	for rows.Next() {
		var hit domain.SearchHit
		if err := rows.Scan(&hit.FormID, &hit.Title, &hit.Snippet, &hit.Status); err != nil {
			return nil, err
		}
		out = append(out, &hit)
	}
	return out, rows.Err()
}

// prefixed — перечень колонок с алиасом таблицы: список полей один, а запросы с
// JOIN требуют квалификации.
func prefixed(cols, alias string) string {
	parts := strings.Split(cols, ",")
	for i, p := range parts {
		parts[i] = alias + "." + strings.TrimSpace(p)
	}
	return strings.Join(parts, ", ")
}
