package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/DmitriyODS/gw2/back-go/pets/internal/domain"
)

// ShopRepo — витрина магазина питомца (pet_shop_items, pet_shop_purchases).
type ShopRepo struct {
	pool *pgxpool.Pool
}

var _ domain.ShopRepo = (*ShopRepo)(nil)

func NewShopRepo(pool *pgxpool.Pool) *ShopRepo {
	return &ShopRepo{pool: pool}
}

const shopItemCols = `id, key, kind, rarity, price_kudos, unlock_kind,
	achievement_key, limited_quota, active_from, active_to`

func scanShopItem(row pgx.Row) (*domain.ShopItem, error) {
	var it domain.ShopItem
	err := row.Scan(&it.ID, &it.Key, &it.Kind, &it.Rarity, &it.PriceKudos,
		&it.UnlockKind, &it.AchievementKey, &it.LimitedQuota, &it.ActiveFrom, &it.ActiveTo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &it, nil
}

// ListActiveItems — товары без окна дат либо с окном, покрывающим now.
func (r *ShopRepo) ListActiveItems(ctx context.Context, now time.Time) ([]*domain.ShopItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+shopItemCols+` FROM pet_shop_items
		WHERE (active_from IS NULL AND active_to IS NULL)
		   OR (active_from <= $1 AND active_to >= $1)
		ORDER BY kind, rarity, price_kudos`, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*domain.ShopItem
	for rows.Next() {
		it, err := scanShopItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *ShopRepo) GetItem(ctx context.Context, key string) (*domain.ShopItem, error) {
	return scanShopItem(r.pool.QueryRow(ctx,
		`SELECT `+shopItemCols+` FROM pet_shop_items WHERE key = $1`, key))
}

func (r *ShopRepo) CountPurchases(ctx context.Context, itemID, companyID int64) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT count(id) FROM pet_shop_purchases WHERE item_id = $1 AND company_id = $2`,
		itemID, companyID).Scan(&count)
	return count, err
}

// shopLockClass — класс advisory-локов покупок магазина (2-ключевая форма
// pg_advisory_xact_lock: класс + item_id), чтобы не пересекаться с локами
// других сервисов в общей БД.
const shopLockClass = 7402

func (r *ShopRepo) RecordPurchase(ctx context.Context, itemID, companyID, userID int64, quota *int) error {
	if quota == nil {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO pet_shop_purchases (item_id, company_id, user_id, purchased_at)
			VALUES ($1, $2, $3, now())`, itemID, companyID, userID)
		return err
	}
	// Лимитированный тираж: COUNT и INSERT под транзакционным advisory-локом
	// товара — иначе два конкурентных покупателя оба видят остаток 1 и
	// перепродают тираж.
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1, $2::int)`,
		shopLockClass, itemID); err != nil {
		return err
	}
	var bought int
	if err := tx.QueryRow(ctx, `
		SELECT count(id) FROM pet_shop_purchases WHERE item_id = $1 AND company_id = $2`,
		itemID, companyID).Scan(&bought); err != nil {
		return err
	}
	if bought >= *quota {
		return domain.ErrSoldOut
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO pet_shop_purchases (item_id, company_id, user_id, purchased_at)
		VALUES ($1, $2, $3, now())`, itemID, companyID, userID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
