package services

import (
	"context"
	"fmt"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidNeedKey se devuelve si una need_key no esta en el conjunto permitido.
type ErrInvalidNeedKey struct{ Key string }

func (e ErrInvalidNeedKey) Error() string { return "invalid need_key: " + e.Key }

type ProfileService struct {
	db          *pgxpool.Pool
	profileRepo *repositories.ProfileRepository
}

func NewProfileService(db *pgxpool.Pool, profileRepo *repositories.ProfileRepository) *ProfileService {
	return &ProfileService{db: db, profileRepo: profileRepo}
}

// Get devuelve las necesidades y el estado de consentimiento del usuario.
func (s *ProfileService) Get(ctx context.Context, userID int64) (*models.ProfileResponse, error) {
	needs, err := s.profileRepo.GetNeeds(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile: needs: %w", err)
	}
	consentAt, err := s.profileRepo.GetConsent(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("profile: consent: %w", err)
	}
	if needs == nil {
		needs = []string{}
	}
	return &models.ProfileResponse{
		Needs:      needs,
		ConsentAt:  consentAt,
		HasConsent: consentAt != nil,
	}, nil
}

// Set fija las necesidades y marca el consentimiento explicito (opt-in). El
// handler solo llega aqui si req.Consent == true.
func (s *ProfileService) Set(ctx context.Context, userID int64, req models.ProfileRequest) (*models.ProfileResponse, error) {
	for _, need := range req.Needs {
		if !models.ValidNeedKeys[need] {
			return nil, ErrInvalidNeedKey{Key: need}
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("profile: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := s.profileRepo.ReplaceNeedsTx(ctx, tx, userID, req.Needs); err != nil {
		return nil, fmt.Errorf("profile: replace needs: %w", err)
	}
	if err := s.profileRepo.SetConsentTx(ctx, tx, userID); err != nil {
		return nil, fmt.Errorf("profile: set consent: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("profile: commit: %w", err)
	}
	return s.Get(ctx, userID)
}

// Delete borra el perfil y retira el consentimiento.
func (s *ProfileService) Delete(ctx context.Context, userID int64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("profile: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := s.profileRepo.ReplaceNeedsTx(ctx, tx, userID, nil); err != nil {
		return fmt.Errorf("profile: clear needs: %w", err)
	}
	if err := s.profileRepo.ClearConsentTx(ctx, tx, userID); err != nil {
		return fmt.Errorf("profile: clear consent: %w", err)
	}
	return tx.Commit(ctx)
}
