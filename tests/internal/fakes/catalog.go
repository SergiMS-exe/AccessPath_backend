package fakes

import (
	"context"
	"errors"

	"accesspath/pkg/catalog"
	"accesspath/pkg/gmaplink"
)

// CatalogProvider es un stub de catalog.Provider.
type CatalogProvider struct {
	SourceFn  func() string
	SearchFn  func(ctx context.Context, query string, limit int) ([]catalog.Item, error)
	DetailsFn func(ctx context.Context, source, externalID string) (*catalog.Item, error)
	NearbyFn  func(ctx context.Context, lat, lng, radiusMeters float64, name string) ([]catalog.Item, error)
}

func (f *CatalogProvider) Source() string {
	if f.SourceFn != nil {
		return f.SourceFn()
	}
	return "overture"
}

func (f *CatalogProvider) Search(ctx context.Context, query string, limit int) ([]catalog.Item, error) {
	if f.SearchFn != nil {
		return f.SearchFn(ctx, query, limit)
	}
	return nil, errors.New("CatalogProvider.Search not stubbed")
}

func (f *CatalogProvider) Details(ctx context.Context, source, externalID string) (*catalog.Item, error) {
	if f.DetailsFn != nil {
		return f.DetailsFn(ctx, source, externalID)
	}
	return nil, errors.New("CatalogProvider.Details not stubbed")
}

func (f *CatalogProvider) Nearby(ctx context.Context, lat, lng, radiusMeters float64, name string) ([]catalog.Item, error) {
	if f.NearbyFn != nil {
		return f.NearbyFn(ctx, lat, lng, radiusMeters, name)
	}
	return nil, errors.New("CatalogProvider.Nearby not stubbed")
}

// LinkResolver es un stub de services.LinkResolver.
type LinkResolver struct {
	ResolveFn func(ctx context.Context, rawURL string) (*gmaplink.Link, error)
}

func (f *LinkResolver) Resolve(ctx context.Context, rawURL string) (*gmaplink.Link, error) {
	if f.ResolveFn != nil {
		return f.ResolveFn(ctx, rawURL)
	}
	return nil, errors.New("LinkResolver.Resolve not stubbed")
}
