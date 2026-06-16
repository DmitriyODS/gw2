package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

// LocationRepo — локации пользователей (user_locations, домен groove).
type LocationRepo struct {
	pool *pgxpool.Pool
}

var _ domain.LocationRepo = (*LocationRepo)(nil)

func NewLocationRepo(pool *pgxpool.Pool) *LocationRepo {
	return &LocationRepo{pool: pool}
}

func (r *LocationRepo) GetLocation(ctx context.Context, userID int64) (*domain.UserLocation, error) {
	var loc domain.UserLocation
	err := r.pool.QueryRow(ctx, `
		SELECT ul.user_id, ul.latitude, ul.longitude, ul.city, ul.updated_at
		FROM user_locations ul
		WHERE ul.user_id = $1`, userID,
	).Scan(&loc.UserID, &loc.Lat, &loc.Lon, &loc.City, &loc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &loc, nil
}

func (r *LocationRepo) SaveLocation(ctx context.Context, loc *domain.UserLocation) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO user_locations (user_id, latitude, longitude, city, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (user_id) DO UPDATE
		SET latitude = $2, longitude = $3, city = $4, updated_at = now()`,
		loc.UserID, loc.Lat, loc.Lon, loc.City)
	return err
}

func (r *LocationRepo) DeleteLocation(ctx context.Context, userID int64) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM user_locations WHERE user_id = $1`, userID)
	return err
}

func (r *LocationRepo) ListLocations(ctx context.Context) ([]*domain.UserLocation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ul.user_id, ul.latitude, ul.longitude, ul.city, ul.updated_at
		FROM user_locations ul
		JOIN users u ON u.id = ul.user_id
		WHERE u.is_active`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.UserLocation
	for rows.Next() {
		var loc domain.UserLocation
		if err := rows.Scan(&loc.UserID, &loc.Lat, &loc.Lon,
			&loc.City, &loc.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, &loc)
	}
	return out, rows.Err()
}
