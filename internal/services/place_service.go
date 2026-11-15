package services

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
	"accesspath/pkg/apperr"
	"accesspath/pkg/catalog"
	"accesspath/pkg/gmaplink"

	"github.com/jackc/pgx/v5"
)

// Errores de dominio del package services. Mantenemos estos sentinels para
// que los services ya existentes puedan seguir usandolos sin cambios; el
// helper apperr.FromService los reconoce y los traduce a respuestas HTTP
// consistentes.
var (
	// ErrPlaceClosedPermanently: el sitio figura como cerrado definitivamente
	// en el catalogo; no tiene sentido importarlo ni valorarlo.
	ErrPlaceClosedPermanently = errors.New("place is permanently closed")
	// ErrCatalogItemNotFound: el (source, external_id) no esta en el catalogo.
	ErrCatalogItemNotFound = errors.New("catalog item not found")
	// ErrCatalogUnavailable: no hay proveedor de catalogo configurado.
	ErrCatalogUnavailable = errors.New("places catalog not configured")
)

const (
	// matchRadiusMeters es la distancia maxima para considerar que un enlace
	// y una ficha del catalogo son el mismo sitio.
	matchRadiusMeters = 50
	// minNameScore es el umbral de similitud de nombre para ese match.
	minNameScore = 0.3
	// communityDedupeMeters es el radio para no crear duplicados comunitarios.
	communityDedupeMeters = 50
	// maxSearchResults acota el buscador (catalogo + lugares propios).
	maxSearchResults = 10
	// moderationPending marca los sitios creados por usuarios.
	moderationPending = "pending"
	// sourceCommunity marca el origen de los sitios creados por usuarios.
	sourceCommunity = "community"
)

// LinkResolver abstrae pkg/gmaplink para poder testear el servicio sin red.
type LinkResolver interface {
	Resolve(ctx context.Context, rawURL string) (*gmaplink.Link, error)
}

type PlaceService interface {
	GetAll(ctx context.Context, filters models.PlaceFilters) (*models.PlaceListResult, error)
	GetByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.PlaceMapItem, error)
	GetNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error)
	GetByID(ctx context.Context, id int64) (*models.PlaceDetail, error)
	Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error)
	Update(ctx context.Context, id, userID int64, req models.UpdatePlaceRequest) (*models.Place, error)
	Delete(ctx context.Context, id, userID int64) error
	Search(ctx context.Context, opts models.SearchOptions) ([]models.PlaceSearchItem, error)
	// ImportFromCatalog devuelve el lugar del catalogo ya copiado a place.
	// Si placeID > 0 se ignora el catalogo y se devuelve ese lugar local
	// (un resultado de busqueda que ya existe en la BD).
	ImportFromCatalog(ctx context.Context, source, externalID string, placeID, userID int64) (*models.Place, error)
	ResolveLink(ctx context.Context, rawURL string) (*models.ResolvedLink, error)
	// CreateCommunity devuelve ademas si el lugar se creo (true) o si ya
	// existia uno parecido (false, con la fila existente).
	CreateCommunity(ctx context.Context, req models.CreateCommunityPlaceRequest, userID int64) (*models.Place, bool, error)
}

type pgPlaceService struct {
	repo          repositories.PlaceRepository
	accSvc        AccessibilityService
	submissionSvc SubmissionService
	catalog       catalog.Provider
	links         LinkResolver
}

func NewPlaceService(
	repo repositories.PlaceRepository,
	accSvc AccessibilityService,
	submissionSvc SubmissionService,
	catalogProvider catalog.Provider,
	linkResolver LinkResolver,
) PlaceService {
	return &pgPlaceService{
		repo:          repo,
		accSvc:        accSvc,
		submissionSvc: submissionSvc,
		catalog:       catalogProvider,
		links:         linkResolver,
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

// --- Buscador --------------------------------------------------------------

// Search busca en el catalogo local (Overture) y despues en los lugares
// comunitarios ya verificados.
func (s *pgPlaceService) Search(ctx context.Context, opts models.SearchOptions) ([]models.PlaceSearchItem, error) {
	if s.catalog == nil {
		return nil, ErrCatalogUnavailable
	}
	if opts.Limit <= 0 {
		opts.Limit = maxSearchResults
	}

	items, err := s.catalog.Search(ctx, opts.Query, opts.Limit)
	if err != nil {
		return nil, apperr.Wrap("places.search", err)
	}

	local, err := s.repo.SearchVerified(ctx, opts.Query, opts.Limit)
	if err != nil {
		return nil, apperr.Wrap("places.search", err)
	}

	out := make([]models.PlaceSearchItem, 0, len(items)+len(local))
	for _, it := range items {
		out = append(out, catalogItemToSearchItem(it, opts))
	}
	for _, p := range local {
		if isRepresented(p, items) {
			continue
		}
		out = append(out, localPlaceToSearchItem(p, opts))
	}
	return out, nil
}

// --- Importacion -----------------------------------------------------------

// ImportFromCatalog copia una ficha del catalogo a nuestra tabla place.
// Idempotente por (source, external_id). Si placeID > 0 devuelve ese lugar
// local sin tocar el catalogo.
func (s *pgPlaceService) ImportFromCatalog(ctx context.Context, source, externalID string, placeID, userID int64) (*models.Place, error) {
	if placeID > 0 {
		place, err := s.repo.FindByID(ctx, placeID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, apperr.NotFound("places.import", "Place")
			}
			return nil, apperr.Wrap("places.import", err)
		}
		return place, nil
	}

	if s.catalog == nil {
		return nil, ErrCatalogUnavailable
	}
	if source == "" {
		source = s.catalog.Source()
	}
	if source != s.catalog.Source() {
		return nil, apperr.BadRequest("places.import", "places.unsupported_source",
			"origen de datos no soportado: "+source)
	}

	existing, err := s.repo.FindBySourceExternalID(ctx, source, externalID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.Wrap("places.import", err)
	}

	item, err := s.catalog.Details(ctx, source, externalID)
	if err != nil {
		if errors.Is(err, catalog.ErrNotFound) {
			return nil, ErrCatalogItemNotFound
		}
		return nil, apperr.Wrap("places.import", err)
	}
	if item.OperatingStatus != nil &&
		strings.EqualFold(*item.OperatingStatus, catalog.OperatingStatusPermanentlyClosed) {
		return nil, ErrPlaceClosedPermanently
	}

	req := models.CreatePlaceRequest{
		Name:       item.Name,
		Address:    item.Address,
		Latitude:   item.Latitude,
		Longitude:  item.Longitude,
		CreatedBy:  userID,
		Source:     &source,
		ExternalID: &item.ExternalID,
	}
	place, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, apperr.Wrap("places.import", err)
	}
	return place, nil
}

// --- Fallback por enlace ---------------------------------------------------

// ResolveLink lee un enlace de Google Maps compartido por el usuario y trata
// de encontrar el sitio en el catalogo. No usa ninguna API de Google: solo
// sigue la redireccion del acortador y parsea la URL final.
func (s *pgPlaceService) ResolveLink(ctx context.Context, rawURL string) (*models.ResolvedLink, error) {
	if s.links == nil {
		return nil, apperr.BadRequest("places.resolve_link", "places.link_disabled",
			"la resolucion de enlaces no esta habilitada")
	}

	link, err := s.links.Resolve(ctx, rawURL)
	if err != nil {
		return nil, linkError("places.resolve_link", err)
	}
	if !link.HasData() {
		return nil, apperr.BadRequest("places.resolve_link", "places.link_without_data",
			"el enlace no contiene un lugar")
	}

	res := &models.ResolvedLink{Matched: false}

	if s.catalog != nil && link.HasCoords() && link.Name != "" {
		candidates, err := s.catalog.Nearby(ctx, *link.Lat, *link.Lng, matchRadiusMeters, link.Name)
		if err != nil {
			return nil, apperr.Wrap("places.resolve_link", err)
		}
		if match, ok := catalog.BestMatch(candidates, *link.Lat, *link.Lng, link.Name,
			matchRadiusMeters, minNameScore); ok {
			item := match.Item
			res.Matched = true
			res.Catalog = &item
			return res, nil
		}
	}

	res.Prefill = &models.PlacePrefill{
		Name:       link.Name,
		Latitude:   link.Lat,
		Longitude:  link.Lng,
		GoogleFTID: link.FTID,
		SourceURL:  link.URL,
	}
	if link.Name == "" {
		res.Message = "el enlace no incluye el nombre del sitio; escribelo tu"
	}
	return res, nil
}

// CreateCommunity crea un sitio pendiente de moderacion. Si ya existe un
// lugar parecido a menos de 50 m, devuelve ese lugar en vez de duplicarlo.
func (s *pgPlaceService) CreateCommunity(ctx context.Context, req models.CreateCommunityPlaceRequest, userID int64) (*models.Place, bool, error) {
	existing, err := s.repo.FindSimilar(ctx, req.Latitude, req.Longitude, communityDedupeMeters, req.Name)
	switch {
	case err == nil:
		return existing, false, nil
	case !errors.Is(err, pgx.ErrNoRows):
		return nil, false, apperr.Wrap("places.community", err)
	}

	address := req.Address
	source := sourceCommunity
	status := moderationPending

	place, err := s.repo.Create(ctx, models.CreatePlaceRequest{
		Name:             req.Name,
		Address:          &address,
		Latitude:         req.Latitude,
		Longitude:        req.Longitude,
		Description:      req.Description,
		CreatedBy:        userID,
		Source:           &source,
		GoogleFTID:       req.GoogleFTID,
		SourceURL:        req.SourceURL,
		ModerationStatus: &status,
	})
	if err != nil {
		return nil, false, apperr.Wrap("places.community", err)
	}
	return place, true, nil
}

// --- Helpers ---------------------------------------------------------------

func catalogItemToSearchItem(it catalog.Item, opts models.SearchOptions) models.PlaceSearchItem {
	item := models.PlaceSearchItem{
		PlaceID:       it.ExternalID,
		Description:   it.Name,
		MainText:      it.Name,
		SecondaryText: it.SecondaryText(),
		Source:        it.Source,
		ExternalID:    it.ExternalID,
		Address:       it.Address,
		Latitude:      &it.Latitude,
		Longitude:     &it.Longitude,
		Category:      it.Category,
	}
	if sec := item.SecondaryText; sec != "" {
		item.Description = it.Name + ", " + sec
	}
	if opts.Latitude != nil && opts.Longitude != nil {
		d := it.DistanceMeters(*opts.Latitude, *opts.Longitude)
		item.DistanceMeters = &d
	}
	return item
}

func localPlaceToSearchItem(p models.Place, opts models.SearchOptions) models.PlaceSearchItem {
	placeID := strconv.FormatInt(p.ID, 10)
	item := models.PlaceSearchItem{
		PlaceID:       placeID,
		Description:   p.Name,
		MainText:      p.Name,
		SecondaryText: derefString(p.Address),
		Source:        p.Source,
		ExternalID:    derefString(p.ExternalID),
		Address:       p.Address,
		Latitude:      &p.Latitude,
		Longitude:     &p.Longitude,
		LocalPlaceID:  &p.ID,
	}
	if item.SecondaryText != "" {
		item.Description = p.Name + ", " + item.SecondaryText
	}
	if opts.Latitude != nil && opts.Longitude != nil {
		d := catalog.HaversineMeters(p.Latitude, p.Longitude, *opts.Latitude, *opts.Longitude)
		item.DistanceMeters = &d
	}
	return item
}

// isRepresented evita duplicar en el buscador un lugar propio que ya sale
// como ficha del catalogo.
func isRepresented(p models.Place, items []catalog.Item) bool {
	for _, it := range items {
		if p.ExternalID != nil && *p.ExternalID == it.ExternalID && p.Source == it.Source {
			return true
		}
		if catalog.NameScore(p.Name, it.Name) >= 0.85 &&
			it.DistanceMeters(p.Latitude, p.Longitude) < communityDedupeMeters {
			return true
		}
	}
	return false
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// linkError traduce los errores de pkg/gmaplink a respuestas HTTP concretas.
func linkError(op string, err error) error {
	switch {
	case errors.Is(err, gmaplink.ErrHostNotAllowed),
		errors.Is(err, gmaplink.ErrUnsupportedURL),
		errors.Is(err, gmaplink.ErrInvalidURL):
		return apperr.BadRequest(op, "places.invalid_link", "el enlace no es de Google Maps")
	case errors.Is(err, gmaplink.ErrNoLinkData):
		return apperr.BadRequest(op, "places.link_without_data", "el enlace no contiene un lugar")
	case errors.Is(err, gmaplink.ErrTooManyRedirect):
		return apperr.BadRequest(op, "places.link_too_many_redirects", "demasiadas redirecciones")
	case errors.Is(err, context.DeadlineExceeded):
		return apperr.BadRequest(op, "places.link_timeout", "el enlace tardo demasiado en responder")
	default:
		return apperr.BadRequest(op, "places.link_error", "no se pudo leer el enlace")
	}
}
