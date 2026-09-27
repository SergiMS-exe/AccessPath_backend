package services

import (
	"context"
	"errors"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
	"accesspath/pkg/apperr"
	"accesspath/pkg/gmaps"

	"github.com/jackc/pgx/v5"
)

// Errores de dominio del package services. Mantenemos estos sentinels para
// que los services ya existentes puedan seguir usandolos sin cambios; el
// helper apperr.FromService los reconoce y los traduce a respuestas HTTP
// consistentes.
var (
	ErrGmapsQuotaExceeded = errors.New("google maps monthly quota exceeded")
	// ErrPlaceClosedPermanently: el sitio esta cerrado para siempre segun Google;
	// no tiene sentido importarlo ni valorarlo.
	ErrPlaceClosedPermanently = errors.New("place is permanently closed")
)

// GmapsClient define las operaciones del cliente de Google Maps que el
// servicio de lugares necesita. La implementacion real vive en pkg/gmaps; el
// interface permite mockearla en tests.
type GmapsClient interface {
	Autocomplete(ctx context.Context, query, sessionToken string) ([]gmaps.AutocompleteItem, error)
	Details(ctx context.Context, placeID, sessionToken string) (*gmaps.PlaceDetails, error)
}

type PlaceService interface {
	GetAll(ctx context.Context, filters models.PlaceFilters) (*models.PlaceListResult, error)
	GetByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.PlaceMapItem, error)
	GetNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error)
	GetByID(ctx context.Context, id int64) (*models.PlaceDetail, error)
	Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error)
	Update(ctx context.Context, id, userID int64, req models.UpdatePlaceRequest) (*models.Place, error)
	Delete(ctx context.Context, id, userID int64) error
	Search(ctx context.Context, query, sessionToken string) ([]models.GoogleAutocompleteItem, error)
	ImportFromGoogle(ctx context.Context, googlePlaceID, sessionToken string, userID int64) (*models.Place, error)
}

type pgPlaceService struct {
	repo          repositories.PlaceRepository
	accSvc        AccessibilityService
	submissionSvc SubmissionService
	gmaps         GmapsClient
	gmapsLog      repositories.GmapsLogRepository
	monthlyLimit  int
}

func NewPlaceService(
	repo repositories.PlaceRepository,
	accSvc AccessibilityService,
	submissionSvc SubmissionService,
	gmapsClient GmapsClient,
	gmapsLog repositories.GmapsLogRepository,
	monthlyLimit int,
) PlaceService {
	return &pgPlaceService{
		repo:          repo,
		accSvc:        accSvc,
		submissionSvc: submissionSvc,
		gmaps:         gmapsClient,
		gmapsLog:      gmapsLog,
		monthlyLimit:  monthlyLimit,
	}
}

var _ PlaceService = (*pgPlaceService)(nil)

func (s *pgPlaceService) GetAll(ctx context.Context, filters models.PlaceFilters) (*models.PlaceListResult, error) {
	places, total, err := s.repo.FindAll(ctx, filters)
	if err != nil {
		return nil, apperr.Wrap("places.list", err)
	}
	return &models.PlaceListResult{
		Places: places,
		Total:  total,
		Limit:  filters.Limit,
		Offset: filters.Offset,
	}, nil
}

// GetByBounds devuelve los lugares del mapa enriquecidos con el estado por
// dimension y el estado global (color del marcador).
func (s *pgPlaceService) GetByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.PlaceMapItem, error) {
	places, err := s.repo.FindByBounds(ctx, filters)
	if err != nil {
		return nil, apperr.Wrap("places.map", err)
	}
	if len(places) == 0 {
		return []models.PlaceMapItem{}, nil
	}

	ids := make([]int64, 0, len(places))
	for _, p := range places {
		ids = append(ids, p.ID)
	}
	rows, err := s.repo.GetAggRowsByPlaceIDs(ctx, ids)
	if err != nil {
		return nil, apperr.Wrap("places.map", err)
	}
	rowsByPlace := map[int64][]models.CriterionAggRow{}
	for _, row := range rows {
		rowsByPlace[row.PlaceID] = append(rowsByPlace[row.PlaceID], row)
	}

	items := make([]models.PlaceMapItem, 0, len(places))
	for _, p := range places {
		dims := s.accSvc.BuildDimensions(rowsByPlace[p.ID])
		// El mapa no necesita el desglose criterio a criterio.
		for i := range dims {
			dims[i].Criteria = nil
		}
		items = append(items, models.PlaceMapItem{
			Place:        p,
			Dimensions:   dims,
			OverallState: s.accSvc.OverallState(dims),
		})
	}
	return items, nil
}

func (s *pgPlaceService) GetNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	places, err := s.repo.FindNearby(ctx, filters)
	if err != nil {
		return nil, apperr.Wrap("places.nearby", err)
	}
	return places, nil
}

// GetByID devuelve el detalle: place + desglose de accesibilidad + submissions.
func (s *pgPlaceService) GetByID(ctx context.Context, id int64) (*models.PlaceDetail, error) {
	place, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("places.detail", "Place")
		}
		return nil, apperr.Wrap("places.detail", err)
	}
	rows, err := s.repo.GetAccessibilityRows(ctx, id)
	if err != nil {
		return nil, apperr.Wrap("places.detail", err)
	}
	submissions, err := s.submissionSvc.GetByPlace(ctx, id)
	if err != nil {
		return nil, apperr.Wrap("places.detail", err)
	}
	if submissions == nil {
		submissions = []models.SubmissionWithDetails{}
	}
	return &models.PlaceDetail{
		Place:         *place,
		Accessibility: s.accSvc.BuildPlaceAccessibility(rows),
		Submissions:   submissions,
	}, nil
}

func (s *pgPlaceService) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	place, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, apperr.Wrap("places.create", err)
	}
	return place, nil
}

func (s *pgPlaceService) Update(ctx context.Context, id, userID int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("places.update", "Place")
		}
		return nil, apperr.Wrap("places.update", err)
	}
	if existing.CreatedBy != userID {
		return nil, ErrNotOwner
	}
	place, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, apperr.Wrap("places.update", err)
	}
	return place, nil
}

func (s *pgPlaceService) Delete(ctx context.Context, id, userID int64) error {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperr.NotFound("places.delete", "Place")
		}
		return apperr.Wrap("places.delete", err)
	}
	if existing.CreatedBy != userID {
		return ErrNotOwner
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Wrap("places.delete", err)
	}
	return nil
}

func (s *pgPlaceService) Search(ctx context.Context, query, sessionToken string) ([]models.GoogleAutocompleteItem, error) {
	if s.gmaps == nil {
		return nil, apperr.GmapsNotConfigured("places.search")
	}
	items, err := s.gmaps.Autocomplete(ctx, query, sessionToken)
	if err != nil {
		// gmaps ya devuelve *AppError tipado; lo pasamos tal cual.
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

func (s *pgPlaceService) ImportFromGoogle(ctx context.Context, googlePlaceID, sessionToken string, userID int64) (*models.Place, error) {
	if s.gmaps == nil {
		return nil, apperr.GmapsNotConfigured("places.import")
	}

	existing, err := s.repo.FindByGooglePlaceID(ctx, googlePlaceID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err //nolint:wrapcheck
	}

	if s.monthlyLimit > 0 {
		count, err := s.gmapsLog.CountThisMonth(ctx)
		if err != nil {
			return nil, apperr.Wrap("places.import", err)
		}
		if count >= s.monthlyLimit {
			return nil, ErrGmapsQuotaExceeded
		}
	}

	details, err := s.gmaps.Details(ctx, googlePlaceID, sessionToken)
	if err != nil {
		return nil, err
	}

	// No importar sitios cerrados permanentemente. La llamada a Details ya
	// consumio cuota, asi que se registra igualmente.
	if details.BusinessStatus == gmaps.BusinessStatusClosedPermanently {
		_ = s.gmapsLog.Log(ctx)
		return nil, ErrPlaceClosedPermanently
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
		return nil, apperr.Wrap("places.import", err)
	}
	_ = s.gmapsLog.Log(ctx)
	return place, nil
}