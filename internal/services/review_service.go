package services

import (
	"context"
	"encoding/base64"
	"fmt"

	"accesspath/internal/models"
	"accesspath/internal/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ReviewService struct {
	db         *pgxpool.Pool
	reviewRepo *repositories.ReviewRepository
	photoRepo  *repositories.PhotoRepository
	ratingSvc  *RatingService
	photoSvc   *PhotoService
}

func NewReviewService(
	db *pgxpool.Pool,
	reviewRepo *repositories.ReviewRepository,
	photoRepo *repositories.PhotoRepository,
	ratingSvc *RatingService,
	photoSvc *PhotoService,
) *ReviewService {
	return &ReviewService{
		db:         db,
		reviewRepo: reviewRepo,
		photoRepo:  photoRepo,
		ratingSvc:  ratingSvc,
		photoSvc:   photoSvc,
	}
}

func (s *ReviewService) GetByPlace(ctx context.Context, placeID int64) ([]models.ReviewWithDetails, error) {
	return s.reviewRepo.FindByPlace(ctx, placeID)
}

func (s *ReviewService) GetByUser(ctx context.Context, userID int64) ([]models.Review, error) {
	return s.reviewRepo.FindByUser(ctx, userID)
}

func (s *ReviewService) GetByID(ctx context.Context, id int64) (*models.Review, error) {
	return s.reviewRepo.FindByID(ctx, id)
}

// Create orchestrates the full review creation within a single DB transaction:
//  1. Insert the review row.
//  2. Upsert every rating via RatingService (which also updates place_rating_cache).
//  3. Upload each photo to MinIO (just before commit to minimise orphan risk).
//  4. Persist photo URLs inside the same transaction.
//  5. Commit. On any error the transaction is rolled back.
func (s *ReviewService) Create(ctx context.Context, req models.CreateReviewRequest) (*models.Review, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("review: begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// 1. Insert review
	review, err := s.reviewRepo.CreateTx(ctx, tx, req)
	if err != nil {
		return nil, fmt.Errorf("review: insert: %w", err)
	}

	// 2. Upsert ratings
	for _, r := range req.Ratings {
		if err := s.ratingSvc.UpsertRating(ctx, tx, review.ID, r.SubcategoryID, r.Score); err != nil {
			return nil, fmt.Errorf("review: rating subcategory %d: %w", r.SubcategoryID, err)
		}
	}

	// 3 & 4. Upload photos to MinIO then register in DB (all DB work is still open)
	for _, b64 := range req.Photos {
		data, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return nil, fmt.Errorf("review: decode photo: %w", err)
		}
		url, err := s.photoSvc.Upload(ctx, data)
		if err != nil {
			return nil, fmt.Errorf("review: photo upload: %w", err)
		}
		if _, err := s.photoRepo.SaveTx(ctx, tx, review.ID, url); err != nil {
			return nil, fmt.Errorf("review: save photo: %w", err)
		}
	}

	// 5. Commit
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("review: commit: %w", err)
	}
	return review, nil
}

func (s *ReviewService) Delete(ctx context.Context, id int64) error {
	return s.reviewRepo.Delete(ctx, id)
}
