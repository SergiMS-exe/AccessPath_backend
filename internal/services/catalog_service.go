package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

// CatalogService expone el catalogo del formulario (dimensiones + criterios + opciones).
type CatalogService interface {
	GetCatalog(ctx context.Context) ([]models.DimensionDetail, error)
}

type pgCatalogService struct {
	repo repositories.CatalogRepository
}

func NewCatalogService(repo repositories.CatalogRepository) CatalogService {
	return &pgCatalogService{repo: repo}
}

var _ CatalogService = (*pgCatalogService)(nil)

func (s *pgCatalogService) GetCatalog(ctx context.Context) ([]models.DimensionDetail, error) {
	return s.repo.GetCatalog(ctx)
}