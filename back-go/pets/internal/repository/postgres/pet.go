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
	p.generation, p.house_owned, p.house_placed, p.house_theme,
	p.house_pet_x, p.house_pet_y,
	p.bank_savings, p.bank_savings_accrued_at, p.bank_loan,
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
	var accessories, unlocked, houseOwned, housePlaced []byte
	var uid *int64
	var fio, avatar *string
	err := row.Scan(&p.UserID, &p.CompanyID, &p.Name, &p.Species, &p.Stage, &p.XP,
		&p.Kudos, &p.Hat, &accessories, &p.FeedStreak, &p.LastFedDate, &p.SickSince,
		&p.Recovery, &p.Personality, &unlocked, &p.QuestDate, &p.QuestKind,
		&p.QuestTarget, &p.QuestProgress, &p.QuestClaimed, &p.AdventureUntil,
		&p.AdventurePlace, &p.Generation, &houseOwned, &housePlaced, &p.HouseTheme,
		&p.HousePetX, &p.HousePetY,
		&p.BankSavings, &p.BankSavingsAccruedAt, &p.BankLoan, &uid, &fio, &avatar)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	p.Accessories = scanStrings(accessories)
	p.UnlockedSpecies = scanStrings(unlocked)
	p.HouseOwned = scanStrings(houseOwned)
	p.HousePlaced = scanHouseItems(housePlaced)
	p.User = userRef(uid, fio, avatar)
	return &p, nil
}

// scanHouseItems — jsonb-массив расстановки; строковую легаси-форму
// («голый ключ» первой итерации) конвертирует UnmarshalJSON HouseItem.
func scanHouseItems(raw []byte) []domain.HouseItem {
	var out []domain.HouseItem
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out)
	}
	if out == nil {
		out = []domain.HouseItem{}
	}
	return out
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
// Поля престижа, домика и банка (generation/house_owned/house_placed/
// bank_savings/bank_savings_accrued_at/bank_loan) сюда намеренно НЕ входят:
// они меняются только своими узкими атомарными методами (Prestige/
// BuyHouseDecor/Append*/SaveHousePlaced/BankRepo) — full-row запись из
// конкурентного действия перетирала бы их устаревшим снимком.
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

// RecallAdventure — досрочный платный возврат: одним UPDATE снимает
// приключение и списывает стоимость (self-join отдаёт старое место, как в
// FinishAdventure); конкурентный ленивый возврат уже занулил поля → 0 строк.
func (r *PetRepo) RecallAdventure(ctx context.Context, userID int64, cost int) (string, bool, error) {
	var place *string
	err := r.pool.QueryRow(ctx, `
		UPDATE pets p SET adventure_until = NULL, adventure_place = NULL,
			kudos = p.kudos - $2
		FROM pets old
		WHERE p.user_id = $1 AND old.user_id = p.user_id
		  AND old.adventure_until IS NOT NULL AND old.kudos >= $2
		RETURNING old.adventure_place`, userID, cost).Scan(&place)
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

// DeletePet — питомец и связанные данные одной транзакцией. pet_strokes,
// pet_shop_purchases и pet_kudos_weekly ссылаются на users, а не на pets —
// каскада от удаления питомца нет, чистим явно; pet_activity_log удаляет
// каскад FK (pets ON DELETE CASCADE).
func (r *PetRepo) DeletePet(ctx context.Context, userID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck // после Commit — no-op

	for _, q := range []string{
		`DELETE FROM pet_strokes WHERE pet_user_id = $1`,
		`DELETE FROM pet_shop_purchases WHERE user_id = $1`,
		`DELETE FROM pet_kudos_weekly WHERE user_id = $1`,
		`DELETE FROM pet_kudos_seasonal WHERE user_id = $1`,
		`DELETE FROM pet_season_claims WHERE user_id = $1`,
		`DELETE FROM pet_kudos_ledger WHERE user_id = $1`,
		`DELETE FROM pets WHERE user_id = $1`,
	} {
		if _, err := tx.Exec(ctx, q, userID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
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

func (r *PetRepo) StrokesTodayByStroker(ctx context.Context, strokerID int64, day time.Time) (map[int64]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pet_user_id, count(id) FROM pet_strokes
		WHERE user_id = $1 AND day = $2
		GROUP BY pet_user_id`,
		strokerID, day)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]int{}
	for rows.Next() {
		var owner int64
		var count int
		if err := rows.Scan(&owner, &count); err != nil {
			return nil, err
		}
		out[owner] = count
	}
	return out, rows.Err()
}

// ─────────────────── престиж (перерождение) ─────────────────────────

// Prestige — атомарное перерождение: гейты (максимальная стадия, здоров,
// не в пути) прямо в WHERE, поэтому гонка двух кликов или конкурентная
// эволюция не дадут двойного инкремента поколения. Эксклюзивный вид
// поколения (если положен) добавляется тем же UPDATE.
func (r *PetRepo) Prestige(ctx context.Context, userID int64, unlockSpecies string) (int, bool, error) {
	unlock := `unlocked_species`
	args := []any{userID, domain.MaxStage}
	if unlockSpecies != "" {
		unlock = `CASE WHEN unlocked_species @> $3::jsonb THEN unlocked_species
			ELSE unlocked_species || $3::jsonb END`
		key, err := json.Marshal([]string{unlockSpecies})
		if err != nil {
			return 0, false, err
		}
		args = append(args, string(key))
	}
	var generation int
	err := r.pool.QueryRow(ctx, `
		UPDATE pets SET generation = generation + 1, stage = 0, xp = 0,
			species = 'egg', unlocked_species = `+unlock+`
		WHERE user_id = $1 AND stage >= $2
		  AND sick_since IS NULL AND adventure_until IS NULL
		RETURNING generation`, args...).Scan(&generation)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return generation, true, nil
}

// ─────────────── сезонный трек (кудосы за квартал) ──────────────────

func (r *PetRepo) AddSeasonalKudos(ctx context.Context, userID int64, season string, amount int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO pet_kudos_seasonal (user_id, season, amount)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, season)
		DO UPDATE SET amount = pet_kudos_seasonal.amount + EXCLUDED.amount`,
		userID, season, amount)
	return err
}

func (r *PetRepo) SeasonalKudos(ctx context.Context, userID int64, season string) (int, error) {
	var amount int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM pet_kudos_seasonal
		WHERE user_id = $1 AND season = $2`, userID, season).Scan(&amount)
	return amount, err
}

func (r *PetRepo) SeasonClaims(ctx context.Context, userID int64, season string) ([]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT threshold FROM pet_season_claims
		WHERE user_id = $1 AND season = $2`, userID, season)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []int
	for rows.Next() {
		var t int
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ClaimSeasonReward — PK(user_id, season, threshold) гарантирует «награда
// один раз»: конкурентный второй клик получит false.
func (r *PetRepo) ClaimSeasonReward(ctx context.Context, userID int64, season string, threshold int) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO pet_season_claims (user_id, season, threshold)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, userID, season, threshold)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ─────────────────────────── домик ──────────────────────────────────

// appendJSONKey — атомарное добавление ключа в jsonb-массив колонки, если
// его там ещё нет (false — уже был; конкурентные вызовы не дублируют).
func (r *PetRepo) appendJSONKey(ctx context.Context, userID int64, column, key string) (bool, error) {
	arr, err := json.Marshal([]string{key})
	if err != nil {
		return false, err
	}
	single, err := json.Marshal(key)
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE pets SET `+column+` = `+column+` || $2::jsonb
		WHERE user_id = $1 AND NOT `+column+` @> $3::jsonb`,
		userID, string(arr), string(single))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PetRepo) AppendAccessory(ctx context.Context, userID int64, key string) (bool, error) {
	return r.appendJSONKey(ctx, userID, "accessories", key)
}

func (r *PetRepo) AppendHouseDecor(ctx context.Context, userID int64, key string) (bool, error) {
	return r.appendJSONKey(ctx, userID, "house_owned", key)
}

// BuyHouseDecor — списание цены и добавление декора одним UPDATE: guard по
// балансу и владению в WHERE, гонка с конкурентным начислением/тратой
// невозможна (false — не хватает кудосов либо уже куплен).
func (r *PetRepo) BuyHouseDecor(ctx context.Context, userID int64, key string, price int) (bool, error) {
	arr, err := json.Marshal([]string{key})
	if err != nil {
		return false, err
	}
	single, err := json.Marshal(key)
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx, `
		UPDATE pets SET kudos = kudos - $2, house_owned = house_owned || $3::jsonb
		WHERE user_id = $1 AND kudos >= $2 AND NOT house_owned @> $4::jsonb`,
		userID, price, string(arr), string(single))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PetRepo) SaveHousePlaced(ctx context.Context, userID int64, placed []domain.HouseItem) error {
	arr, err := json.Marshal(placed)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `UPDATE pets SET house_placed = $2 WHERE user_id = $1`,
		userID, string(arr))
	return err
}

func (r *PetRepo) SaveHouseTheme(ctx context.Context, userID int64, theme string) error {
	_, err := r.pool.Exec(ctx, `UPDATE pets SET house_theme = $2 WHERE user_id = $1`,
		userID, theme)
	return err
}

func (r *PetRepo) SaveHousePetPos(ctx context.Context, userID int64, x, y float64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE pets SET house_pet_x = $2, house_pet_y = $3 WHERE user_id = $1`,
		userID, x, y)
	return err
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
