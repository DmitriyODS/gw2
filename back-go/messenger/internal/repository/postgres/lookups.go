package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/DmitriyODS/gw2/back-go/messenger/internal/domain"
)

// Read-only лукапы чужих таблиц (владельцы — callsvc, Flask, groove).

func (r *Repo) GetCall(ctx context.Context, id int64) (*domain.CallInfo, error) {
	var c domain.CallInfo
	err := r.q(ctx).QueryRow(ctx, `
		SELECT id, kind, media, status, started_at, ended_at, initiator_id, conversation_id
		FROM calls WHERE id = $1`, id,
	).Scan(&c.ID, &c.Kind, &c.Media, &c.Status, &c.StartedAt, &c.EndedAt,
		&c.InitiatorID, &c.ConversationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *Repo) GetTask(ctx context.Context, id int64) (*domain.TaskPreview, error) {
	var t domain.TaskPreview
	err := r.q(ctx).QueryRow(ctx, `
		SELECT t.id, t.name, t.is_archived, t.color, tu.fio, t.deadline, t.company_id
		FROM tasks t
		LEFT JOIN users tu ON tu.id = t.responsible_user_id
		WHERE t.id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.IsArchived, &t.Color, &t.ResponsibleFIO, &t.Deadline, &t.CompanyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func (r *Repo) PetName(ctx context.Context, ownerID int64) (*string, error) {
	var name string
	err := r.q(ctx).QueryRow(ctx, `SELECT name FROM pets WHERE user_id = $1`, ownerID).Scan(&name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &name, nil
}
