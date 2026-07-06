package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

// CatalogService expone el catalogo del formulario (dimensiones + criterios + opciones).
type CatalogService struct {
	repo *repositories.CatalogRepository
}

func NewCatalogService(repo *repositories.CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) GetCatalog(ctx context.Context) ([]models.DimensionDetail, error) {
	return s.repo.GetCatalog(ctx)
}
