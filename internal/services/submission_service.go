package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrEmptySubmission se devuelve si no hay comentario ni fotos: no hay nada que envolver.
var ErrEmptySubmission = errors.New("submission requires a comment or at least one photo")

type SubmissionService struct {
	db             *pgxpool.Pool
	submissionRepo *repositories.SubmissionRepository
	photoRepo      *repositories.PhotoRepository
	photoSvc       *PhotoService
}

func NewSubmissionService(
	db *pgxpool.Pool,
	submissionRepo *repositories.SubmissionRepository,
	photoRepo *repositories.PhotoRepository,
	photoSvc *PhotoService,
) *SubmissionService {
	return &SubmissionService{
		db:             db,
		submissionRepo: submissionRepo,
		photoRepo:      photoRepo,
		photoSvc:       photoSvc,
	}
}

// Save hace get-or-create de la valoracion viva del usuario para el lugar, fija
// el comentario (si viene) y sube las fotos adjuntas, todo en una TX. Semantica
// idempotente de PUT: la submission puede existir ya (creada al contribuir).
func (s *SubmissionService) Save(ctx context.Context, userID int64, req models.SubmissionRequest) (*models.Submission, error) {
	hasComment := req.Comment != nil && strings.TrimSpace(*req.Comment) != ""
	if !hasComment && len(req.Photos) == 0 {
		return nil, ErrEmptySubmission
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("submission: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	submission, err := s.submissionRepo.GetOrCreateLiveTx(ctx, tx, userID, req.PlaceID)
	if err != nil {
		return nil, fmt.Errorf("submission: get-or-create: %w", err)
	}

	if hasComment {
		if err := s.submissionRepo.SetCommentTx(ctx, tx, submission.ID, req.Comment); err != nil {
			return nil, fmt.Errorf("submission: set comment: %w", err)
		}
		submission.Comment = req.Comment
	}

	for _, ph := range req.Photos {
		data, err := base64.StdEncoding.DecodeString(ph.Data)
		if err != nil {
			return nil, fmt.Errorf("submission: decode photo: %w", err)
		}
		url, objectKey, err := s.photoSvc.Upload(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("submission: photo upload: %w", err)
		}
		if _, err := s.photoRepo.SaveTx(ctx, tx, submission.ID, ph.ContributionID, url, &objectKey, ph.SuggestedSlot); err != nil {
			return nil, fmt.Errorf("submission: save photo: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("submission: commit: %w", err)
	}
	return submission, nil
}

// GetByPlace devuelve los comentarios + fotos de un lugar ("que cuenta la gente").
func (s *SubmissionService) GetByPlace(ctx context.Context, placeID int64) ([]models.SubmissionWithDetails, error) {
	submissions, err := s.submissionRepo.FindByPlace(ctx, placeID)
	if err != nil {
		return nil, err
	}
	if len(submissions) == 0 {
		return submissions, nil
	}

	ids := make([]int64, 0, len(submissions))
	for _, sub := range submissions {
		ids = append(ids, sub.ID)
	}
	photos, err := s.photoRepo.FindBySubmissionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	photosBySub := map[int64][]models.Photo{}
	for _, p := range photos {
		photosBySub[p.SubmissionID] = append(photosBySub[p.SubmissionID], p)
	}
	for i := range submissions {
		submissions[i].Photos = photosBySub[submissions[i].ID]
	}
	return submissions, nil
}
