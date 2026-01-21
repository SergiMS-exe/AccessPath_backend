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

func (r *UserRepository) FindByID(ctx context.Context, id string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		"SELECT id, email, name, avatar_url, created_at, updated_at FROM users WHERE id = $1", id).
		Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.UserWithPassword, error) {
	var u models.UserWithPassword
	err := r.db.QueryRow(ctx,
		"SELECT id, email, password_hash, name, avatar_url, created_at, updated_at FROM users WHERE email = $1", email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, email, passwordHash, name string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRow(ctx,
		"INSERT INTO users (email, password_hash, name) VALUES ($1, $2, $3) RETURNING id, email, name, avatar_url, created_at, updated_at",
		email, passwordHash, name).
		Scan(&u.ID, &u.Email, &u.Name, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetSavedPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	query := `SELECT p.id, p.google_place_id, p.name, p.address, p.city, p.country, p.latitude, p.longitude, p.place_type, p.created_at, p.updated_at
			  FROM places p
			  JOIN user_saved_places usp ON p.id = usp.place_id
			  WHERE usp.user_id = $1
			  ORDER BY usp.created_at DESC`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []models.Place
	for rows.Next() {
		var p models.Place
		if err := rows.Scan(&p.ID, &p.GooglePlaceID, &p.Name, &p.Address, &p.City, &p.Country, &p.Latitude, &p.Longitude, &p.PlaceType, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		places = append(places, p)
	}

	return places, nil
}

func (r *UserRepository) SavePlace(ctx context.Context, userID, placeID string) error {
	_, err := r.db.Exec(ctx,
		"INSERT INTO user_saved_places (user_id, place_id) VALUES ($1, $2) ON CONFLICT DO NOTHING",
		userID, placeID)
	return err
}

func (r *UserRepository) UnsavePlace(ctx context.Context, userID, placeID string) error {
	_, err := r.db.Exec(ctx,
		"DELETE FROM user_saved_places WHERE user_id = $1 AND place_id = $2",
		userID, placeID)
	return err
}
