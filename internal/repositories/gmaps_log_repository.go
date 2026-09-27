package repositories

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GmapsLogRepository interface {
	CountThisMonth(ctx context.Context) (int, error)
	Log(ctx context.Context) error
}

type pgGmapsLogRepository struct {
	db *pgxpool.Pool
}

func NewGmapsLogRepository(db *pgxpool.Pool) GmapsLogRepository {
	return &pgGmapsLogRepository{db: db}
}

var _ GmapsLogRepository = (*pgGmapsLogRepository)(nil)

func (r *pgGmapsLogRepository) CountThisMonth(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM gmaps_api_log
		 WHERE called_at >= date_trunc('month', NOW())`).
		Scan(&count)
	return count, err
}

func (r *pgGmapsLogRepository) Log(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `INSERT INTO gmaps_api_log DEFAULT VALUES`)
	return err
}