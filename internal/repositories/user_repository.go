package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, username, email, created_at, updated_at, deleted_at
		 FROM "user" WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByCode(ctx context.Context, code string) (*models.User, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, username, email, created_at, updated_at, deleted_at
		 FROM "user" WHERE code = $1 AND deleted_at IS NULL`, code)
	if err != nil {
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.UserWithPassword, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, username, email, password_hash, created_at, updated_at, deleted_at
		 FROM "user" WHERE email = $1 AND deleted_at IS NULL`, email)
	if err != nil {
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.UserWithPassword])
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, req models.CreateUserRequest, passwordHash string) (*models.User, error) {
	rows, err := r.db.Query(ctx,
		`INSERT INTO "user" (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, code, username, email, created_at, updated_at, deleted_at`,
		req.Username, req.Email, passwordHash)
	if err != nil {
		return nil, err
	}
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.User])
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE "user" SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
