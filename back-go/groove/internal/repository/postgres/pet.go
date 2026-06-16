package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/groove/internal/domain"
)

type PetRepo struct {
	pool *pgxpool.Pool
}

var _ domain.PetRepo = (*PetRepo)(nil)

func NewPetRepo(pool *pgxpool.Pool) *PetRepo {
	return &PetRepo{pool: pool}
}

const petCols = `p.user_id, p.company_id, p.name, p.species, p.stage, p.xp, p.beans,
	p.hat, p.accessories, p.feed_streak, p.last_fed_date, p.sick_since, p.recovery,
	p.personality, p.unlocked_species, p.quest_date, p.quest_kind, p.quest_target,
	p.quest_progress, p.quest_claimed, u.id, u.fio, u.avatar_path`

const petFrom = ` FROM pets p LEFT JOIN users u ON u.id = p.user_id `

func scanStrings(raw []byte) []string {
	var out []string
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func scanPet(row pgx.Row) (*domain.Pet, error) {
	var p domain.Pet
	var accessories, unlocked []byte
	var uid *int64
	var fio, avatar *string
	err := row.Scan(&p.UserID, &p.CompanyID, &p.Name, &p.Species, &p.Stage, &p.XP,
		&p.Beans, &p.Hat, &accessories, &p.FeedStreak, &p.LastFedDate, &p.SickSince,
		&p.Recovery, &p.Personality, &unlocked, &p.QuestDate, &p.QuestKind,
		&p.QuestTarget, &p.QuestProgress, &p.QuestClaimed, &uid, &fio, &avatar)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.Accessories = scanStrings(accessories)
	p.UnlockedSpecies = scanStrings(unlocked)
	p.User = userRef(uid, fio, avatar)
	return &p, nil
}

func (r *PetRepo) GetPet(ctx context.Context, userID int64) (*domain.Pet, error) {
	return scanPet(r.pool.QueryRow(ctx,
		`SELECT `+petCols+petFrom+`WHERE p.user_id = $1`, userID))
}

func (r *PetRepo) GetOrCreate(ctx context.Context, userID, companyID int64) (*domain.Pet, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pets (user_id, company_id, name, species, stage, xp, beans,
		                  accessories, feed_streak, recovery, unlocked_species,
		                  quest_progress, quest_claimed, created_at)
		VALUES ($1, $2, 'Грувик', 'egg', 0, 0, 0, '[]', 0, 0, '[]', 0, FALSE, now())
		ON CONFLICT (user_id) DO NOTHING`, userID, companyID)
	if err != nil {
		return nil, err
	}
	return r.GetPet(ctx, userID)
}

// SavePet — полное сохранение изменяемых полей (по образу ORM-коммита Flask).
func (r *PetRepo) SavePet(ctx context.Context, p *domain.Pet) error {
	accessories, err := json.Marshal(p.Accessories)
	if err != nil {
		return err
	}
	unlocked, err := json.Marshal(p.UnlockedSpecies)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE pets SET name = $2, species = $3, stage = $4, xp = $5, beans = $6,
			hat = $7, accessories = $8, feed_streak = $9, last_fed_date = $10,
			sick_since = $11, recovery = $12, personality = $13,
			unlocked_species = $14, quest_date = $15, quest_kind = $16,
			quest_target = $17, quest_progress = $18, quest_claimed = $19
		WHERE user_id = $1`,
		p.UserID, p.Name, p.Species, p.Stage, p.XP, p.Beans, p.Hat, accessories,
		p.FeedStreak, p.LastFedDate, p.SickSince, p.Recovery, p.Personality,
		unlocked, p.QuestDate, p.QuestKind, p.QuestTarget, p.QuestProgress,
		p.QuestClaimed)
	return err
}

func (r *PetRepo) ListCompanyPets(ctx context.Context, companyID int64) ([]*domain.Pet, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+petCols+`
		FROM pets p JOIN users u ON u.id = p.user_id
		WHERE p.company_id = $1 AND u.is_active
		ORDER BY p.stage DESC, p.xp DESC`, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.Pet
	for rows.Next() {
		p, err := scanPet(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PetRepo) LastUnitEndByUsers(ctx context.Context, userIDs []int64) (map[int64]time.Time, error) {
	result := map[int64]time.Time{}
	if len(userIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, max(datetime_end) FROM units
		WHERE user_id = ANY($1) AND datetime_end IS NOT NULL
		GROUP BY user_id`, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var userID int64
		var lastEnd time.Time
		if err := rows.Scan(&userID, &lastEnd); err != nil {
			return nil, err
		}
		result[userID] = lastEnd
	}
	return result, rows.Err()
}

func (r *PetRepo) SoulmateForUser(ctx context.Context, userID int64,
	since time.Time) (*domain.UserRef, int, error) {

	var ref domain.UserRef
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, u.fio, u.avatar_path, count(un.id) AS cnt
		FROM units un
		JOIN users u ON u.id = un.user_id
		WHERE un.task_id IN (
			SELECT DISTINCT task_id FROM units
			WHERE user_id = $1 AND datetime_start >= $2
		) AND un.user_id != $1 AND un.datetime_start >= $2 AND u.is_active
		GROUP BY u.id, u.fio, u.avatar_path
		ORDER BY cnt DESC
		LIMIT 1`, userID, since).Scan(&ref.ID, &ref.FIO, &ref.AvatarPath, &count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, 0, nil
		}
		return nil, 0, err
	}
	return &ref, count, nil
}

func (r *PetRepo) FinishedUnitsForUser(ctx context.Context, userID int64,
	since time.Time, limit int) ([]domain.FinishedUnit, error) {

	rows, err := r.pool.Query(ctx, `
		SELECT name, datetime_start, datetime_end FROM units
		WHERE user_id = $1 AND datetime_end IS NOT NULL AND datetime_start >= $2
		ORDER BY datetime_start DESC
		LIMIT $3`, userID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.FinishedUnit
	for rows.Next() {
		var u domain.FinishedUnit
		if err := rows.Scan(&u.Name, &u.Start, &u.End); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// ──────────────────────────── поглаживания ─────────────────────────

func (r *PetRepo) AddStroke(ctx context.Context, petUserID, userID int64, day time.Time) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO pet_strokes (pet_user_id, user_id, day, created_at)
		VALUES ($1, $2, $3, now())
		ON CONFLICT (pet_user_id, user_id, day) DO NOTHING`,
		petUserID, userID, day)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (r *PetRepo) StrokesToday(ctx context.Context, petUserIDs []int64, day time.Time) (map[int64]int, error) {
	result := map[int64]int{}
	if len(petUserIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT pet_user_id, count(id) FROM pet_strokes
		WHERE pet_user_id = ANY($1) AND day = $2
		GROUP BY pet_user_id`, petUserIDs, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var petUserID int64
		var count int
		if err := rows.Scan(&petUserID, &count); err != nil {
			return nil, err
		}
		result[petUserID] = count
	}
	return result, rows.Err()
}

func (r *PetRepo) MyStrokesToday(ctx context.Context, userID int64, day time.Time) (map[int64]bool, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pet_user_id FROM pet_strokes WHERE user_id = $1 AND day = $2`,
		userID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int64]bool{}
	for rows.Next() {
		var petUserID int64
		if err := rows.Scan(&petUserID); err != nil {
			return nil, err
		}
		result[petUserID] = true
	}
	return result, rows.Err()
}

// ────────────────────────────── рейды ──────────────────────────────

func scanRaid(row pgx.Row) (*domain.Raid, error) {
	var raid domain.Raid
	err := row.Scan(&raid.ID, &raid.CompanyID, &raid.WeekStart, &raid.Boss,
		&raid.Target, &raid.Reward, &raid.DefeatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &raid, nil
}

const raidCols = `id, company_id, week_start, boss, target, reward, defeated_at`

func (r *PetRepo) GetRaid(ctx context.Context, companyID int64, weekStart time.Time) (*domain.Raid, error) {
	return scanRaid(r.pool.QueryRow(ctx, `
		SELECT `+raidCols+` FROM groove_raids
		WHERE company_id = $1 AND week_start = $2`, companyID, weekStart))
}

func (r *PetRepo) CreateRaid(ctx context.Context, companyID int64, weekStart time.Time,
	boss string, target int, reward string) (*domain.Raid, error) {

	return scanRaid(r.pool.QueryRow(ctx, `
		INSERT INTO groove_raids (company_id, week_start, boss, target, reward, created_at)
		VALUES ($1, $2, $3, $4, $5, now())
		RETURNING `+raidCols, companyID, weekStart, boss, target, reward))
}

func (r *PetRepo) SetRaidDefeated(ctx context.Context, raidID int64, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE groove_raids SET defeated_at = $2 WHERE id = $1`, raidID, at)
	return err
}

func (r *PetRepo) GrantRaidRewards(ctx context.Context, companyID int64,
	beans int, reward string) error {

	_, err := r.pool.Exec(ctx, `
		UPDATE pets SET beans = beans + $2,
			accessories = CASE
				WHEN accessories @> to_jsonb($3::text) THEN accessories
				ELSE accessories || to_jsonb($3::text)
			END
		WHERE company_id = $1`, companyID, beans, reward)
	return err
}

func (r *PetRepo) CountClosedBetween(ctx context.Context, companyID int64,
	start, end time.Time) (int, error) {

	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(id) FROM tasks
		WHERE company_id = $1 AND is_archived = TRUE
		  AND archived_at >= $2 AND archived_at < $3`,
		companyID, start, end).Scan(&count)
	return count, err
}
