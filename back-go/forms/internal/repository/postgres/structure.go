package postgres

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/forms/internal/domain"
)

const sectionCols = `id, form_id, title, description, position, next_action,
	next_section_id, visible_question_id, visible_values, created_at`

const questionCols = `id, form_id, section_id, type, title, description, required,
	config, position, points, answer_key, created_at`

// ListSections — разделы формы вместе с вопросами: два запроса на форму, а не
// запрос на раздел.
func (r *Repo) ListSections(ctx context.Context, formID int64) ([]domain.Section, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+sectionCols+` FROM form_sections WHERE form_id = $1 ORDER BY position, id`,
		formID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sections := []domain.Section{}
	index := map[int64]int{}
	for rows.Next() {
		var s domain.Section
		var visible []string
		if err := rows.Scan(&s.ID, &s.FormID, &s.Title, &s.Description, &s.Position,
			&s.NextAction, &s.NextSectionID, &s.VisibleQuestionID, &visible, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.VisibleValues = visible
		s.NextIndex = -1
		s.Questions = []domain.Question{}
		index[s.ID] = len(sections)
		sections = append(sections, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(sections) == 0 {
		return sections, nil
	}

	qRows, err := r.pool.Query(ctx,
		`SELECT `+questionCols+` FROM form_questions WHERE form_id = $1 ORDER BY position, id`,
		formID)
	if err != nil {
		return nil, err
	}
	defer qRows.Close()
	for qRows.Next() {
		q, err := scanQuestion(qRows)
		if err != nil {
			return nil, err
		}
		if i, ok := index[q.SectionID]; ok {
			sections[i].Questions = append(sections[i].Questions, q)
		}
	}
	return sections, qRows.Err()
}

func scanQuestion(row pgx.Row) (domain.Question, error) {
	var q domain.Question
	err := row.Scan(&q.ID, &q.FormID, &q.SectionID, &q.Type, &q.Title, &q.Description,
		&q.Required, &q.Config, &q.Position, &q.Points, &q.AnswerKey, &q.CreatedAt)
	if q.Config == nil {
		q.Config = map[string]any{}
	}
	if q.AnswerKey == nil {
		q.AnswerKey = map[string]any{}
	}
	return q, err
}

func (r *Repo) GetQuestion(ctx context.Context, formID, questionID int64) (*domain.Question, error) {
	q, err := scanQuestion(r.pool.QueryRow(ctx,
		`SELECT `+questionCols+` FROM form_questions WHERE id = $1 AND form_id = $2`,
		questionID, formID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &q, nil
}

/*
ReplaceStructure — синхронизация разделов и вопросов формы в одной транзакции.

	Ссылки ветвления приезжают ПОЗИЦИЯМИ разделов (Section.NextIndex и значения
	config.targets вида "#2"), потому что у только что добавленного раздела id
	появляется лишь здесь; в id их переводим сами, когда все разделы на месте.

	Порядок операций важен: сначала разделы, потом вопросы (вопрос могли
	перенести в другой раздел), и только потом удаление лишнего — иначе каскад
	от удалённого раздела унёс бы перенесённый вопрос.
*/
func (r *Repo) ReplaceStructure(ctx context.Context, formID int64, sections []domain.Section) ([]int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	existingSections, err := idsOf(ctx, tx, `SELECT id FROM form_sections WHERE form_id = $1`, formID)
	if err != nil {
		return nil, err
	}
	existingQuestions, err := idsOf(ctx, tx, `SELECT id FROM form_questions WHERE form_id = $1`, formID)
	if err != nil {
		return nil, err
	}

	// ── Разделы ──
	keepSections := map[int64]bool{}
	idByIndex := make([]int64, len(sections))
	for i := range sections {
		s := &sections[i]
		s.Position = i
		if s.NextAction != domain.NextSection && s.NextAction != domain.NextSubmit {
			s.NextAction = domain.NextNext
		}
		if s.ID > 0 && existingSections[s.ID] {
			keepSections[s.ID] = true
			if _, err := tx.Exec(ctx,
				`UPDATE form_sections
				    SET title=$2, description=$3, position=$4, next_action=$5,
				        visible_question_id=$6, visible_values=$7
				  WHERE id=$1`,
				s.ID, s.Title, s.Description, s.Position, s.NextAction,
				s.VisibleQuestionID, visibleValues(s.VisibleValues)); err != nil {
				return nil, err
			}
		} else {
			if err := tx.QueryRow(ctx,
				`INSERT INTO form_sections
				   (form_id, title, description, position, next_action,
				    visible_question_id, visible_values)
				 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id, created_at`,
				formID, s.Title, s.Description, s.Position, s.NextAction,
				s.VisibleQuestionID, visibleValues(s.VisibleValues)).
				Scan(&s.ID, &s.CreatedAt); err != nil {
				return nil, err
			}
			keepSections[s.ID] = true
		}
		s.FormID = formID
		idByIndex[i] = s.ID
	}

	// Переходы разделов — вторым проходом: цель могла быть создана после самого
	// раздела, и до этого момента её id не существовало.
	for i := range sections {
		s := &sections[i]
		var next *int64
		if s.NextAction == domain.NextSection {
			if s.NextIndex >= 0 && s.NextIndex < len(idByIndex) {
				id := idByIndex[s.NextIndex]
				next = &id
			} else if s.NextSectionID != nil && keepSections[*s.NextSectionID] {
				next = s.NextSectionID
			}
		}
		s.NextSectionID = next
		if _, err := tx.Exec(ctx,
			`UPDATE form_sections SET next_section_id = $2 WHERE id = $1`, s.ID, next); err != nil {
			return nil, err
		}
	}

	// ── Вопросы ──
	keepQuestions := map[int64]bool{}
	position := 0
	for i := range sections {
		s := &sections[i]
		for j := range s.Questions {
			q := &s.Questions[j]
			q.FormID, q.SectionID, q.Position = formID, s.ID, position
			position++
			q.Config = resolveTargets(q.Config, idByIndex)
			if q.ID > 0 && existingQuestions[q.ID] {
				keepQuestions[q.ID] = true
				if _, err := tx.Exec(ctx,
					`UPDATE form_questions
					    SET section_id=$2, type=$3, title=$4, description=$5, required=$6,
					        config=$7, position=$8, points=$9, answer_key=$10
					  WHERE id=$1`,
					q.ID, q.SectionID, q.Type, q.Title, q.Description, q.Required,
					q.Config, q.Position, q.Points, q.AnswerKey); err != nil {
					return nil, err
				}
				continue
			}
			if err := tx.QueryRow(ctx,
				`INSERT INTO form_questions
				   (form_id, section_id, type, title, description, required, config,
				    position, points, answer_key)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id, created_at`,
				formID, q.SectionID, q.Type, q.Title, q.Description, q.Required, q.Config,
				q.Position, q.Points, q.AnswerKey).Scan(&q.ID, &q.CreatedAt); err != nil {
				return nil, err
			}
			keepQuestions[q.ID] = true
		}
	}

	// ── Удаление лишнего ──
	removed := []int64{}
	for id := range existingQuestions {
		if !keepQuestions[id] {
			removed = append(removed, id)
		}
	}
	if len(removed) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM form_questions WHERE id = ANY($1)`, removed); err != nil {
			return nil, err
		}
	}
	staleSections := []int64{}
	for id := range existingSections {
		if !keepSections[id] {
			staleSections = append(staleSections, id)
		}
	}
	if len(staleSections) > 0 {
		if _, err := tx.Exec(ctx, `DELETE FROM form_sections WHERE id = ANY($1)`, staleSections); err != nil {
			return nil, err
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE forms SET updated_at = now() WHERE id = $1`, formID); err != nil {
		return nil, err
	}
	return removed, tx.Commit(ctx)
}

// resolveTargets — переходы вопроса из позиций разделов в их id. Значения
// "submit" и "next" остаются как есть, недостижимая позиция — отбрасывается.
func resolveTargets(config map[string]any, idByIndex []int64) map[string]any {
	raw, ok := config["targets"].(map[string]any)
	if !ok || len(raw) == 0 {
		return config
	}
	out := make(map[string]any, len(config))
	for k, v := range config {
		out[k] = v
	}
	targets := make(map[string]any, len(raw))
	for option, target := range raw {
		s := domain.Text(target)
		switch {
		case s == domain.NextSubmit || s == domain.NextNext:
			targets[option] = s
		default:
			if i, ok := domain.ParseTargetIndex(s); ok {
				if i < len(idByIndex) {
					targets[option] = strconv.FormatInt(idByIndex[i], 10)
				}
				continue
			}
			// Уже идентификатор раздела (форма сохраняется без правки ветвления).
			if _, err := strconv.ParseInt(s, 10, 64); err == nil {
				targets[option] = s
			}
		}
	}
	out["targets"] = targets
	return out
}

// visibleValues — ожидаемые ответы условия показа в виде, который принимает
// JSONB-колонка (nil там означал бы null, а не «условия нет»).
func visibleValues(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// idsOf — множество id по запросу (какие разделы и вопросы формы уже есть).
func idsOf(ctx context.Context, tx pgx.Tx, sql string, args ...any) (map[int64]bool, error) {
	rows, err := tx.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}
