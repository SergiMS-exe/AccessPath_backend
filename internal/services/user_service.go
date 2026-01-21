package services

import (
	"context"
	"errors"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo *repositories.UserRepository
}

func NewUserService(repo *repositories.UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetByID(ctx context.Context, id string) (*models.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *UserService) Register(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	// Check if user already exists
	existing, _ := s.repo.FindByEmail(ctx, req.Email)
	if existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return s.repo.Create(ctx, req.Email, string(hashedPassword), req.Name)
}

func (s *UserService) Login(ctx context.Context, req models.LoginRequest) (*models.User, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	return &user.User, nil
}

func (s *UserService) GetSavedPlaces(ctx context.Context, userID string) ([]models.Place, error) {
	return s.repo.GetSavedPlaces(ctx, userID)
}

func (s *UserService) SavePlace(ctx context.Context, userID, placeID string) error {
	return s.repo.SavePlace(ctx, userID, placeID)
}

func (s *UserService) UnsavePlace(ctx context.Context, userID, placeID string) error {
	return s.repo.UnsavePlace(ctx, userID, placeID)
}
