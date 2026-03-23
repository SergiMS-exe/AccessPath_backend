package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`SELECT id, code, username, email, created_at, updated_at, deleted_at
		 FROM users WHERE id = $1 AND deleted_at IS NULL`, id).
		Scan(&u.ID, &u.Code, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByCode(ctx context.Context, code string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`SELECT id, code, username, email, created_at, updated_at, deleted_at
		 FROM users WHERE code = $1 AND deleted_at IS NULL`, code).
		Scan(&u.ID, &u.Code, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	err := r.db.QueryRow(ctx,
		`SELECT id, code, username, email, password_hash, created_at, updated_at, deleted_at
		 FROM users WHERE email = $1 AND deleted_at IS NULL`, email).
		Scan(&u.ID, &u.Code, &u.Username, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, req models.CreateUserRequest, passwordHash string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		`INSERT INTO users (username, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, code, username, email, created_at, updated_at, deleted_at`,
		req.Username, req.Email, passwordHash).
		Scan(&u.ID, &u.Code, &u.Username, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET deleted_at = NOW() WHERE id = $1`, id)
	return err
}
