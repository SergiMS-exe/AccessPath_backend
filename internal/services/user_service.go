package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

// Sentinels de dominio de UserService. apperr.FromService los traduce a la
// respuesta HTTP adecuada (401 / 400).
var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailAlreadyUsed   = errors.New("email already registered")
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id int64) (*models.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) Register(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	existing, _ := s.repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailAlreadyUsed
	}

	// Si el cliente no manda username, generamos uno a partir del email:
	// "juan" para juan@example.com, "juan.7f3c" si ya existe colision.
	if strings.TrimSpace(req.Username) == "" {
		req.Username = generateUsernameFromEmail(req.Email)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, req, string(hashedPassword))
}

func (s *UserService) Login(ctx context.Context, req models.LoginRequest) (*models.User, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &user.User, nil
}

// generateUsernameFromEmail toma la parte local del email (antes del @), la
// sanea (solo [a-z0-9._-], en minusculas) y, si ya existe en BD, le pega un
// sufijo corto derivado del hash del email completo para mantenerlo unico.
func generateUsernameFromEmail(email string) string {
	local := strings.ToLower(strings.SplitN(email, "@", 2)[0])
	var b strings.Builder
	for _, r := range local {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune('.')
		}
	}
	base := strings.Trim(b.String(), ".")
	if base == "" {
		base = "user"
	}
	return base + "." + shortEmailFingerprint(email)
}

// shortEmailFingerprint devuelve 4 chars hex del hash del email. Suficiente
// para que dos emails distintos generen usernames distintos sin colisionar
// en BD (que ademas tiene UNIQUE en email y en username).
func shortEmailFingerprint(email string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(email)))
	return hex.EncodeToString(sum[:])[:4]
}
