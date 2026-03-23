package services

import (
	"context"

	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type CategoryService struct {
	repo *repositories.CategoryRepository
}

func NewCategoryService(repo *repositories.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) GetAllCategories(ctx context.Context) ([]models.Category, error) {
	return s.repo.FindAllCategories(ctx)
}

func (s *CategoryService) GetCategoryByID(ctx context.Context, id int64) (*models.Category, error) {
	return s.repo.FindCategoryByID(ctx, id)
}

func (s *CategoryService) CreateCategory(ctx context.Context, req models.CreateCategoryRequest) (*models.Category, error) {
	return s.repo.CreateCategory(ctx, req)
}

func (s *CategoryService) UpdateCategory(ctx context.Context, id int64, req models.CreateCategoryRequest) (*models.Category, error) {
	return s.repo.UpdateCategory(ctx, id, req)
}

func (s *CategoryService) GetAllSubcategories(ctx context.Context) ([]models.SubcategoryWithCategory, error) {
	return s.repo.FindAllSubcategories(ctx)
}

func (s *CategoryService) GetSubcategoriesByCategory(ctx context.Context, categoryID int64) ([]models.Subcategory, error) {
	return s.repo.FindSubcategoriesByCategory(ctx, categoryID)
}

func (s *CategoryService) GetSubcategoryByID(ctx context.Context, id int64) (*models.SubcategoryWithCategory, error) {
	return s.repo.FindSubcategoryByID(ctx, id)
}

func (s *CategoryService) CreateSubcategory(ctx context.Context, req models.CreateSubcategoryRequest) (*models.Subcategory, error) {
	return s.repo.CreateSubcategory(ctx, req)
}

func (s *CategoryService) UpdateSubcategory(ctx context.Context, id int64, req models.CreateSubcategoryRequest) (*models.Subcategory, error) {
	return s.repo.UpdateSubcategory(ctx, id, req)
}
