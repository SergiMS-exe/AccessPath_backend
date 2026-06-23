package services

import (
	"context"
	"errors"
	"fmt"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
	"accesspath/pkg/gmaps"

	"github.com/jackc/pgx/v5"
)

var ErrGmapsQuotaExceeded = errors.New("google maps monthly quota exceeded")

type PlaceService struct {
	repo         *repositories.PlaceRepository
	ratingSvc    *RatingService
	gmaps        *gmaps.Client
	gmapsLog     *repositories.GmapsLogRepository
	monthlyLimit int
}

func NewPlaceService(repo *repositories.PlaceRepository, ratingSvc *RatingService, gmapsClient *gmaps.Client, gmapsLog *repositories.GmapsLogRepository, monthlyLimit int) *PlaceService {
	return &PlaceService{repo: repo, ratingSvc: ratingSvc, gmaps: gmapsClient, gmapsLog: gmapsLog, monthlyLimit: monthlyLimit}
}

func (s *PlaceService) GetAll(ctx context.Context, filters models.PlaceFilters) (*models.PlaceListResult, error) {
	places, total, err := s.repo.FindAll(ctx, filters)
	if err != nil {
		return nil, err
	}
	return &models.PlaceListResult{
		Places: places,
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}, nil
}

func (s *PlaceService) GetByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.Place, error) {
	return s.repo.FindByBounds(ctx, filters)
}

func (s *PlaceService) GetNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	return s.repo.FindNearby(ctx, filters)
}

func (s *PlaceService) GetByID(ctx context.Context, id int64) (*models.PlaceDetail, error) {
	place, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("place: find: %w", err)
	}
	ratings, err := s.ratingSvc.GetPlaceRatings(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("place: ratings: %w", err)
	}
	return &models.PlaceDetail{Place: *place, Ratings: ratings}, nil
}

func (s *PlaceService) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	return s.repo.Create(ctx, req)
}

func (s *PlaceService) Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *PlaceService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *PlaceService) Search(ctx context.Context, query, sessionToken string) ([]models.GoogleAutocompleteItem, error) {
	if s.gmaps == nil {
		return nil, fmt.Errorf("google maps not configured")
	}
	items, err := s.gmaps.Autocomplete(ctx, query, sessionToken)
	if err != nil {
		return nil, err
	}
	result := make([]models.GoogleAutocompleteItem, 0, len(items))
	for _, it := range items {
		result = append(result, models.GoogleAutocompleteItem{
			PlaceID:       it.PlaceID,
			Description:   it.Description,
			MainText:      it.MainText,
			SecondaryText: it.SecondaryText,
		})
	}
	return result, nil
}

func (s *PlaceService) ImportFromGoogle(ctx context.Context, googlePlaceID, sessionToken string, userID int64) (*models.Place, error) {
	if s.gmaps == nil {
		return nil, fmt.Errorf("google maps not configured")
	}

	existing, err := s.repo.FindByGooglePlaceID(ctx, googlePlaceID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("import: check existing: %w", err)
	}

	if s.monthlyLimit > 0 {
		count, err := s.gmapsLog.CountThisMonth(ctx)
		if err != nil {
			return nil, fmt.Errorf("import: check quota: %w", err)
		}
		if count >= s.monthlyLimit {
			return nil, ErrGmapsQuotaExceeded
		}
	}

	details, err := s.gmaps.Details(ctx, googlePlaceID, sessionToken)
	if err != nil {
		return nil, fmt.Errorf("import: google details: %w", err)
	}

	req := models.CreatePlaceRequest{
		Name:          details.Name,
		Address:       &details.FormattedAddress,
		Latitude:      details.Lat,
		Longitude:     details.Lng,
		GooglePlaceID: &details.PlaceID,
		CreatedBy:     userID,
	}
	place, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	_ = s.gmapsLog.Log(ctx)
	return place, nil
}
