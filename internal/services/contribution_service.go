package services

import (
	"context"
	"errors"
	"fmt"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrContributionNotFound = errors.New("contribution not found")
	ErrNotOwner             = errors.New("not the owner")
	ErrOptionMismatch       = errors.New("answer option does not belong to criterion")
)

type ContributionService struct {
	db             *pgxpool.Pool
	contribRepo    *repositories.ContributionRepository
	submissionRepo *repositories.SubmissionRepository
	catalogRepo    *repositories.CatalogRepository
	placeRepo      *repositories.PlaceRepository
	accSvc         *AccessibilityService
}

func NewContributionService(
	db *pgxpool.Pool,
	contribRepo *repositories.ContributionRepository,
	submissionRepo *repositories.SubmissionRepository,
	catalogRepo *repositories.CatalogRepository,
	placeRepo *repositories.PlaceRepository,
	accSvc *AccessibilityService,
) *ContributionService {
	return &ContributionService{
		db:             db,
		contribRepo:    contribRepo,
		submissionRepo: submissionRepo,
		catalogRepo:    catalogRepo,
		placeRepo:      placeRepo,
		accSvc:         accSvc,
	}
}

// Create resuelve exists/quality desde la opcion, hace upsert de la contribucion
// viva, publica el lugar si es la primera y recalcula el cache del (place,criterion),
// todo en una sola transaccion. Devuelve el semaforo en vivo.
func (s *ContributionService) Create(ctx context.Context, userID int64, req models.ContributionRequest) (*models.ContributionResult, error) {
	opt, err := s.catalogRepo.GetOptionByID(ctx, req.AnswerOptionID)
	if err != nil {
		return nil, fmt.Errorf("contribution: option: %w", err)
	}
	if opt.CriterionID != req.CriterionID {
		return nil, ErrOptionMismatch
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("contribution: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// La contribucion siempre cuelga de la valoracion viva del usuario para el
	// lugar: get-or-create de la submission antes de guardar la respuesta.
	submission, err := s.submissionRepo.GetOrCreateLiveTx(ctx, tx, userID, req.PlaceID)
	if err != nil {
		return nil, fmt.Errorf("contribution: get-or-create submission: %w", err)
	}

	contribution, err := s.contribRepo.UpsertLiveTx(ctx, tx, submission.ID, userID, req.PlaceID, req.CriterionID, req.AnswerOptionID, opt.ExistsValue, opt.QualityValue)
	if err != nil {
		return nil, fmt.Errorf("contribution: upsert: %w", err)
	}

	if err := s.placeRepo.MarkPublishedTx(ctx, tx, req.PlaceID); err != nil {
		return nil, fmt.Errorf("contribution: publish place: %w", err)
	}

	if err := s.contribRepo.RecalculateCacheTx(ctx, tx, req.PlaceID, req.CriterionID); err != nil {
		return nil, fmt.Errorf("contribution: recalc cache: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("contribution: commit: %w", err)
	}

	result, err := s.liveResult(ctx, req.PlaceID, req.CriterionID)
	if err != nil {
		return nil, err
	}
	result.ContributionID = &contribution.ID
	return result, nil
}

// Delete valida propiedad, hace soft delete (Deshacer) y recalcula el cache.
func (s *ContributionService) Delete(ctx context.Context, userID, id int64) (*models.ContributionResult, error) {
	contribution, err := s.contribRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrContributionNotFound
	}
	if contribution.UserID != userID {
		return nil, ErrNotOwner
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("contribution: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := s.contribRepo.SoftDeleteTx(ctx, tx, id); err != nil {
		return nil, fmt.Errorf("contribution: soft delete: %w", err)
	}
	if err := s.contribRepo.RecalculateCacheTx(ctx, tx, contribution.PlaceID, contribution.CriterionID); err != nil {
		return nil, fmt.Errorf("contribution: recalc cache: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("contribution: commit: %w", err)
	}

	return s.liveResult(ctx, contribution.PlaceID, contribution.CriterionID)
}

// liveResult construye el estado nuevo del criterio y su dimension tras el cambio.
func (s *ContributionService) liveResult(ctx context.Context, placeID, criterionID int64) (*models.ContributionResult, error) {
	rows, err := s.placeRepo.GetAccessibilityRows(ctx, placeID)
	if err != nil {
		return nil, fmt.Errorf("contribution: accessibility rows: %w", err)
	}
	dims := s.accSvc.BuildDimensions(rows)

	for _, dim := range dims {
		for _, cr := range dim.Criteria {
			if cr.CriterionID == criterionID {
				// La dimension del resultado no arrastra el desglose completo.
				dimSummary := dim
				dimSummary.Criteria = nil
				return &models.ContributionResult{
					Criterion: cr,
					Dimension: dimSummary,
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("contribution: criterion %d not found in catalog", criterionID)
}
