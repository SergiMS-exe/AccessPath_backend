package services_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"accesspath/internal/models"
	"accesspath/internal/services"
	"accesspath/pkg/apperr"
	"accesspath/pkg/catalog"
	"accesspath/pkg/gmaplink"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"accesspath/tests/internal/fakes"
)

// --- Search con catalogo ---------------------------------------------------

func TestPlaceService_Search_UsesCatalogAndLocal(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{
		SearchFn: func(_ context.Context, q string, limit int) ([]catalog.Item, error) {
			assert.Equal(t, "eclipse", q)
			assert.Equal(t, 10, limit, "sin limit explicito se usa maxSearchResults")
			return []catalog.Item{{
				Source:     "overture",
				ExternalID: "gers-1",
				Name:       "Cafeteria Eclipse",
				Address:    strPtr("Calle Leopoldo Lugones, 24"),
				City:       strPtr("Lugones"),
				Latitude:   43.4021,
				Longitude:  -5.804946,
			}}, nil
		},
	}
	d.repo.SearchVerifiedFn = func(_ context.Context, _ string, _ int) ([]models.Place, error) {
		return []models.Place{{
			ID:               7,
			Name:             "Bar Propio",
			Address:          strPtr("Calle Otro, 1"),
			Latitude:         43.4,
			Longitude:        -5.8,
			Source:           "community",
			ModerationStatus: "verified",
		}}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.Search(context.Background(), models.SearchOptions{Query: "eclipse"})
	require.NoError(t, err)
	require.Len(t, got, 2)

	assert.Equal(t, "gers-1", got[0].ExternalID)
	assert.Equal(t, "overture", got[0].Source)
	assert.Equal(t, "Cafeteria Eclipse", got[0].MainText)
	assert.Equal(t, "Calle Leopoldo Lugones, 24, Lugones", got[0].SecondaryText)
	assert.Contains(t, got[0].Description, "Cafeteria Eclipse")

	assert.Equal(t, "7", got[1].PlaceID, "un lugar propio se identifica por id local")
	require.NotNil(t, got[1].LocalPlaceID)
	assert.Equal(t, int64(7), *got[1].LocalPlaceID)
}

func TestPlaceService_Search_DedupesLocalCopyOfCatalogItem(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{
		SearchFn: func(_ context.Context, _ string, _ int) ([]catalog.Item, error) {
			return []catalog.Item{{
				Source: "overture", ExternalID: "gers-1", Name: "Cafeteria Eclipse",
				Latitude: 43.4021, Longitude: -5.804946,
			}}, nil
		},
	}
	d.repo.SearchVerifiedFn = func(_ context.Context, _ string, _ int) ([]models.Place, error) {
		return []models.Place{{
			ID: 9, Name: "Cafeteria Eclipse", Source: "overture", ExternalID: strPtr("gers-1"),
			Latitude: 43.4021, Longitude: -5.804946,
		}}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.Search(context.Background(), models.SearchOptions{Query: "eclipse"})
	require.NoError(t, err)
	assert.Len(t, got, 1, "el lugar ya importado no se duplica")
}

func TestPlaceService_Search_CatalogErrorIs500(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{
		SearchFn: func(_ context.Context, _ string, _ int) ([]catalog.Item, error) {
			return nil, errors.New("db down")
		},
	}
	svc := newPlaceService(d, 0)

	_, err := svc.Search(context.Background(), models.SearchOptions{Query: "x"})
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusInternalServerError, ae.HTTPStatus)
}

// --- ImportFromCatalog -----------------------------------------------------

func TestPlaceService_ImportFromCatalog_ReturnsExisting(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{}
	d.repo.FindBySourceExternalFn = func(_ context.Context, source, id string) (*models.Place, error) {
		assert.Equal(t, "overture", source)
		assert.Equal(t, "gers-1", id)
		return &models.Place{ID: 3, Name: "Ya importado"}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.ImportFromCatalog(context.Background(), "", "gers-1", 0, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(3), got.ID)
}

func TestPlaceService_ImportFromCatalog_NotFound(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{
		DetailsFn: func(_ context.Context, _, _ string) (*catalog.Item, error) {
			return nil, catalog.ErrNotFound
		},
	}
	d.repo.FindBySourceExternalFn = func(_ context.Context, _, _ string) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newPlaceService(d, 0)

	_, err := svc.ImportFromCatalog(context.Background(), "overture", "nope", 0, 42)
	require.ErrorIs(t, err, services.ErrCatalogItemNotFound)
}

func TestPlaceService_ImportFromCatalog_PermanentlyClosed(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{
		DetailsFn: func(_ context.Context, _, _ string) (*catalog.Item, error) {
			return &catalog.Item{ExternalID: "x", Name: "Cerrado", OperatingStatus: strPtr("permanently_closed")}, nil
		},
	}
	d.repo.FindBySourceExternalFn = func(_ context.Context, _, _ string) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newPlaceService(d, 0)

	_, err := svc.ImportFromCatalog(context.Background(), "overture", "x", 0, 42)
	require.ErrorIs(t, err, services.ErrPlaceClosedPermanently)
}

func TestPlaceService_ImportFromCatalog_CopiesFields(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{
		DetailsFn: func(_ context.Context, source, id string) (*catalog.Item, error) {
			assert.Equal(t, "overture", source)
			assert.Equal(t, "gers-1", id)
			return &catalog.Item{
				Source: "overture", ExternalID: "gers-1", Name: "Cafeteria Eclipse",
				Address:  strPtr("Calle Leopoldo Lugones, 24"),
				Latitude: 43.4021, Longitude: -5.804946,
			}, nil
		},
	}
	d.repo.FindBySourceExternalFn = func(_ context.Context, _, _ string) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	var captured models.CreatePlaceRequest
	d.repo.CreateFn = func(_ context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
		captured = req
		return &models.Place{ID: 11, Name: req.Name}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.ImportFromCatalog(context.Background(), "", "gers-1", 0, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(11), got.ID)
	assert.Equal(t, "Cafeteria Eclipse", captured.Name)
	assert.Equal(t, int64(42), captured.CreatedBy)
	require.NotNil(t, captured.Source)
	assert.Equal(t, "overture", *captured.Source)
	require.NotNil(t, captured.ExternalID)
	assert.Equal(t, "gers-1", *captured.ExternalID)
	assert.Nil(t, captured.ModerationStatus, "lo importado del catalogo se considera verificado")
}

func TestPlaceService_ImportFromCatalog_ByLocalID(t *testing.T) {
	d := newPlaceDeps()
	// Sin catalogo: el modo place_id no lo necesita.
	d.repo.FindByIDFn = func(_ context.Context, id int64) (*models.Place, error) {
		assert.Equal(t, int64(12), id)
		return &models.Place{ID: 12, Name: "Local"}, nil
	}
	svc := newPlaceService(d, 0)

	got, err := svc.ImportFromCatalog(context.Background(), "", "", 12, 42)
	require.NoError(t, err)
	assert.Equal(t, int64(12), got.ID)
}

func TestPlaceService_ImportFromCatalog_ByLocalIDNotFound(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindByIDFn = func(_ context.Context, _ int64) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	svc := newPlaceService(d, 0)

	_, err := svc.ImportFromCatalog(context.Background(), "", "", 99, 42)
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusNotFound, ae.HTTPStatus)
}

func TestPlaceService_ImportFromCatalog_RejectsOtherSource(t *testing.T) {
	d := newPlaceDeps()
	d.catalog = &fakes.CatalogProvider{}
	svc := newPlaceService(d, 0)

	_, err := svc.ImportFromCatalog(context.Background(), "foursquare", "x", 0, 42)
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, "places.unsupported_source", ae.Code)
}

// --- ResolveLink -----------------------------------------------------------

func TestPlaceService_ResolveLink_MatchesCatalog(t *testing.T) {
	d := newPlaceDeps()
	d.links = &fakes.LinkResolver{
		ResolveFn: func(_ context.Context, raw string) (*gmaplink.Link, error) {
			assert.Contains(t, raw, "maps.app.goo.gl")
			return &gmaplink.Link{
				URL:  "https://www.google.com/maps/place/Cafeteria+Eclipse/...",
				Kind: gmaplink.KindPlace,
				Name: "Cafeteria Eclipse",
				Lat:  floatPtr(43.4020665),
				Lng:  floatPtr(-5.804991),
				FTID: "0xd368bf0484e32bf:0xb9556be5462221b7",
			}, nil
		},
	}
	d.catalog = &fakes.CatalogProvider{
		NearbyFn: func(_ context.Context, lat, lng, radius float64, name string) ([]catalog.Item, error) {
			assert.InDelta(t, 43.4020665, lat, 1e-9)
			assert.InDelta(t, 50.0, radius, 1e-9)
			assert.Equal(t, "Cafeteria Eclipse", name)
			return []catalog.Item{{
				Source: "overture", ExternalID: "gers-1", Name: "Cafetería Eclipse",
				Address:  strPtr("Calle Leopoldo Lugones, 24"),
				Latitude: 43.4021, Longitude: -5.804946,
			}}, nil
		},
	}
	svc := newPlaceService(d, 0)

	got, err := svc.ResolveLink(context.Background(), "https://maps.app.goo.gl/43FKHoBREWRVfGwe8")
	require.NoError(t, err)
	assert.True(t, got.Matched)
	require.NotNil(t, got.Catalog)
	assert.Equal(t, "gers-1", got.Catalog.ExternalID)
	assert.Nil(t, got.Prefill)
}

func TestPlaceService_ResolveLink_NoMatchReturnsPrefill(t *testing.T) {
	d := newPlaceDeps()
	d.links = &fakes.LinkResolver{
		ResolveFn: func(_ context.Context, _ string) (*gmaplink.Link, error) {
			return &gmaplink.Link{
				URL:  "https://www.google.com/maps/place/Bar+Nuevo/...",
				Kind: gmaplink.KindPlace, Name: "Bar Nuevo",
				Lat: floatPtr(43.4), Lng: floatPtr(-5.8),
				FTID: "0xaa:0xbb",
			}, nil
		},
	}
	d.catalog = &fakes.CatalogProvider{
		NearbyFn: func(_ context.Context, _, _, _ float64, _ string) ([]catalog.Item, error) {
			// 200 m de distancia: no es el mismo sitio.
			return []catalog.Item{{
				Source: "overture", ExternalID: "otro", Name: "Bar Nuevo",
				Latitude: 43.4018, Longitude: -5.8,
			}}, nil
		},
	}
	svc := newPlaceService(d, 0)

	got, err := svc.ResolveLink(context.Background(), "https://www.google.com/maps/place/Bar+Nuevo/")
	require.NoError(t, err)
	assert.False(t, got.Matched)
	assert.Nil(t, got.Catalog)
	require.NotNil(t, got.Prefill)
	assert.Equal(t, "Bar Nuevo", got.Prefill.Name)
	assert.Equal(t, "0xaa:0xbb", got.Prefill.GoogleFTID)
}

func TestPlaceService_ResolveLink_RejectsNonGoogleLink(t *testing.T) {
	d := newPlaceDeps()
	d.links = &fakes.LinkResolver{
		ResolveFn: func(_ context.Context, _ string) (*gmaplink.Link, error) {
			return nil, gmaplink.ErrHostNotAllowed
		},
	}
	svc := newPlaceService(d, 0)

	_, err := svc.ResolveLink(context.Background(), "https://evil.example.com/maps")
	require.Error(t, err)
	var ae *apperr.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, http.StatusBadRequest, ae.HTTPStatus)
	assert.Equal(t, "places.invalid_link", ae.Code)
}

// --- CreateCommunity -------------------------------------------------------

func TestPlaceService_CreateCommunity_CreatesPending(t *testing.T) {
	d := newPlaceDeps()
	d.repo.FindSimilarFn = func(_ context.Context, _, _, _ float64, _ string) (*models.Place, error) {
		return nil, pgx.ErrNoRows
	}
	var captured models.CreatePlaceRequest
	d.repo.CreateFn = func(_ context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
		captured = req
		return &models.Place{ID: 21, Name: req.Name, ModerationStatus: "pending", Source: "community"}, nil
	}
	svc := newPlaceService(d, 0)

	got, created, err := svc.CreateCommunity(context.Background(), models.CreateCommunityPlaceRequest{
		Name: "Bar Nuevo", Address: "Calle X, 1", Latitude: 43.4, Longitude: -5.8,
	}, 42)

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, int64(21), got.ID)
	assert.Equal(t, int64(42), captured.CreatedBy)
	require.NotNil(t, captured.Source)
	assert.Equal(t, "community", *captured.Source)
	require.NotNil(t, captured.ModerationStatus)
	assert.Equal(t, "pending", *captured.ModerationStatus)
	require.NotNil(t, captured.Address)
	assert.Equal(t, "Calle X, 1", *captured.Address)
}

func TestPlaceService_CreateCommunity_ReturnsExistingSimilarPlace(t *testing.T) {
	d := newPlaceDeps()
	existing := &models.Place{ID: 22, Name: "Bar Nuevo"}
	d.repo.FindSimilarFn = func(_ context.Context, _, _, _ float64, _ string) (*models.Place, error) {
		return existing, nil
	}
	svc := newPlaceService(d, 0)

	got, created, err := svc.CreateCommunity(context.Background(), models.CreateCommunityPlaceRequest{
		Name: "Bar Nuevo", Address: "Calle X, 1", Latitude: 43.4, Longitude: -5.8,
	}, 42)

	require.NoError(t, err)
	assert.False(t, created, "no se duplica: se devuelve el existente")
	assert.Equal(t, int64(22), got.ID)
}

// --- helpers ---------------------------------------------------------------

func strPtr(s string) *string { return &s }
