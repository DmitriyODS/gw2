package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

type PetRepo struct {
	pool *pgxpool.Pool
}

var _ domain.PetRepo = (*PetRepo)(nil)

func NewPetRepo(pool *pgxpool.Pool) *PetRepo {
	return &PetRepo{pool: pool}
}

const petCols = `p.user_id, p.company_id, p.name, p.species, p.stage, p.xp, p.kudos,
	p.hat, p.accessories, p.feed_streak, p.last_fed_date, p.sick_since, p.recovery,
	p.personality, p.unlocked_species, p.quest_date, p.quest_kind, p.quest_target,
	p.quest_progress, p.quest_claimed, p.adventure_until, p.adventure_place,
	u.id, u.fio, u.avatar_path`

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
		&p.Kudos, &p.Hat, &accessories, &p.FeedStreak, &p.LastFedDate, &p.SickSince,
		&p.Recovery, &p.Personality, &unlocked, &p.QuestDate, &p.QuestKind,
		&p.QuestTarget, &p.QuestProgress, &p.QuestClaimed, &p.AdventureUntil,
		&p.AdventurePlace, &uid, &fio, &avatar)
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
		INSERT INTO pets (user_id, company_id, name, species, stage, xp, kudos,
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
		UPDATE pets SET name = $2, species = $3, stage = $4, xp = $5, kudos = $6,
			hat = $7, accessories = $8, feed_streak = $9, last_fed_date = $10,
			sick_since = $11, recovery = $12, personality = $13,
			unlocked_species = $14, quest_date = $15, quest_kind = $16,
			quest_target = $17, quest_progress = $18, quest_claimed = $19,
			adventure_until = $20, adventure_place = $21
		WHERE user_id = $1`,
		p.UserID, p.Name, p.Species, p.Stage, p.XP, p.Kudos, p.Hat, accessories,
		p.FeedStreak, p.LastFedDate, p.SickSince, p.Recovery, p.Personality,
		unlocked, p.QuestDate, p.QuestKind, p.QuestTarget, p.QuestProgress,
		p.QuestClaimed, p.AdventureUntil, p.AdventurePlace)
	return err
}

// ──────────────────────────── приключение ───────────────────────────

// StartAdventure — узкий UPDATE полей приключения с guard'ом в SQL: гонка
// двух стартов (или старт больного) отдаёт false без ошибки.
func (r *PetRepo) StartAdventure(ctx context.Context, userID int64, until time.Time, place string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		UPDATE pets SET adventure_until = $2, adventure_place = $3
		WHERE user_id = $1 AND adventure_until IS NULL AND sick_since IS NULL`,
		userID, until, place)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// FinishAdventure — атомарный возврат: NULL-ит поля приключения, только если
// срок истёк. Self-join отдаёт СТАРОЕ значение adventure_place (RETURNING
// после SET вернул бы NULL); конкурентный второй GET перепроверит предикат
// по обновлённой строке и получит 0 строк — награда начисляется один раз.
func (r *PetRepo) FinishAdventure(ctx context.Context, userID int64, now time.Time) (string, bool, error) {
	var place *string
	err := r.pool.QueryRow(ctx, `
		UPDATE pets p SET adventure_until = NULL, adventure_place = NULL
		FROM pets old
		WHERE p.user_id = $1 AND old.user_id = p.user_id
		  AND old.adventure_until IS NOT NULL AND old.adventure_until <= $2
		RETURNING old.adventure_place`, userID, now).Scan(&place)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, nil
		}
		return "", false, err
	}
	if place == nil {
		return "", true, nil
	}
	return *place, true, nil
}

// AdjustBalances — атомарный инкремент балансов: конкурентные начисления
// (хук юнита, покупка, кормление) не перетирают друг друга, в отличие от
// full-row SavePet.
func (r *PetRepo) AdjustBalances(ctx context.Context, userID int64, deltaKudos, deltaXP int) (int, int, error) {
	var kudos, xp int
	err := r.pool.QueryRow(ctx, `
		UPDATE pets SET kudos = GREATEST(0, kudos + $2), xp = xp + $3
		WHERE user_id = $1
		RETURNING kudos, xp`, userID, deltaKudos, deltaXP).Scan(&kudos, &xp)
	return kudos, xp, err
}

// SaveEvolution — только поля эволюции; балансы/квест не трогаем, чтобы не
// затереть конкурентные инкременты.
func (r *PetRepo) SaveEvolution(ctx context.Context, p *domain.Pet) error {
	unlocked, err := json.Marshal(p.UnlockedSpecies)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE pets SET stage = $2, species = $3, personality = $4, unlocked_species = $5
		WHERE user_id = $1`,
		p.UserID, p.Stage, p.Species, p.Personality, unlocked)
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

// ─────────────────── рейтинг признания (кудосы за неделю) ──────────

func (r *PetRepo) AddWeeklyKudos(ctx context.Context, userID int64, isoYear, isoWeek, amount int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pet_kudos_weekly (user_id, iso_year, iso_week, amount)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, iso_year, iso_week)
		DO UPDATE SET amount = pet_kudos_weekly.amount + EXCLUDED.amount`,
		userID, isoYear, isoWeek, amount)
	return err
}

func (r *PetRepo) WeeklyKudosCounts(ctx context.Context, companyID int64, isoYear, isoWeek int) (map[int64]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT w.user_id, w.amount
		FROM pet_kudos_weekly w
		JOIN pets p ON p.user_id = w.user_id
		WHERE p.company_id = $1 AND w.iso_year = $2 AND w.iso_week = $3`,
		companyID, isoYear, isoWeek)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int64]int{}
	for rows.Next() {
		var userID int64
		var amount int
		if err := rows.Scan(&userID, &amount); err != nil {
			return nil, err
		}
		result[userID] = amount
	}
	return result, rows.Err()
}

// ───────────────────────── поглаживание ─────────────────────────────

func (r *PetRepo) StrokesToday(ctx context.Context, petOwnerID, strokerID int64, day time.Time) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(id) FROM pet_strokes
		WHERE pet_user_id = $1 AND user_id = $2 AND day = $3`,
		petOwnerID, strokerID, day).Scan(&count)
	return count, err
}

// RecordStroke — один INSERT на поглаживание (не upsert): дневной лимит
// StrokeDailyMaxPerPet считается по количеству строк за день, поэтому
// старое ограничение UNIQUE(pet_user_id, user_id, day) снято миграцией
// (см. 000NN_pets_restructure.sql) в пользу обычного индекса.
func (r *PetRepo) RecordStroke(ctx context.Context, petOwnerID, strokerID int64, day time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pet_strokes (pet_user_id, user_id, day, created_at)
		VALUES ($1, $2, $3, now())`,
		petOwnerID, strokerID, day)
	return err
}
