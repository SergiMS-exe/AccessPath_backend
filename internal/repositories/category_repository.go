package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	db *pgxpool.Pool
}

func NewCategoryRepository(db *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// --- Categories ---

func (r *CategoryRepository) FindAllCategories(ctx context.Context) ([]models.Category, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, name, slug, is_active, display_order, created_at, updated_at
		 FROM categories
		 ORDER BY display_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var c models.Category
		if err := rows.Scan(&c.ID, &c.Code, &c.Name, &c.Slug, &c.IsActive, &c.DisplayOrder,
			&c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}
	return categories, nil
}

func (r *CategoryRepository) FindCategoryByID(ctx context.Context, id int64) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRow(ctx,
		`SELECT id, code, name, slug, is_active, display_order, created_at, updated_at
		 FROM categories WHERE id = $1`, id).
		Scan(&c.ID, &c.Code, &c.Name, &c.Slug, &c.IsActive, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) CreateCategory(ctx context.Context, req models.CreateCategoryRequest) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRow(ctx,
		`INSERT INTO categories (name, slug, is_active, display_order)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, code, name, slug, is_active, display_order, created_at, updated_at`,
		req.Name, req.Slug, req.IsActive, req.DisplayOrder).
		Scan(&c.ID, &c.Code, &c.Name, &c.Slug, &c.IsActive, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *CategoryRepository) UpdateCategory(ctx context.Context, id int64, req models.CreateCategoryRequest) (*models.Category, error) {
	var c models.Category
	err := r.db.QueryRow(ctx,
		`UPDATE categories
		 SET name = $2, slug = $3, is_active = $4, display_order = $5, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, code, name, slug, is_active, display_order, created_at, updated_at`,
		id, req.Name, req.Slug, req.IsActive, req.DisplayOrder).
		Scan(&c.ID, &c.Code, &c.Name, &c.Slug, &c.IsActive, &c.DisplayOrder, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// --- Subcategories ---

func (r *CategoryRepository) FindAllSubcategories(ctx context.Context) ([]models.SubcategoryWithCategory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT s.id, s.code, s.category_id, s.name, s.slug, s.is_active, s.display_order, s.created_at, s.updated_at,
		        c.name AS category_name
		 FROM subcategories s
		 JOIN categories c ON s.category_id = c.id
		 ORDER BY c.display_order, s.display_order, s.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.SubcategoryWithCategory
	for rows.Next() {
		var s models.SubcategoryWithCategory
		if err := rows.Scan(&s.ID, &s.Code, &s.CategoryID, &s.Name, &s.Slug, &s.IsActive, &s.DisplayOrder,
			&s.CreatedAt, &s.UpdatedAt, &s.CategoryName); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *CategoryRepository) FindSubcategoriesByCategory(ctx context.Context, categoryID int64) ([]models.Subcategory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, code, category_id, name, slug, is_active, display_order, created_at, updated_at
		 FROM subcategories
		 WHERE category_id = $1
		 ORDER BY display_order, id`, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subcategory
	for rows.Next() {
		var s models.Subcategory
		if err := rows.Scan(&s.ID, &s.Code, &s.CategoryID, &s.Name, &s.Slug, &s.IsActive, &s.DisplayOrder,
			&s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}

func (r *CategoryRepository) FindSubcategoryByID(ctx context.Context, id int64) (*models.SubcategoryWithCategory, error) {
	var s models.SubcategoryWithCategory
	err := r.db.QueryRow(ctx,
		`SELECT s.id, s.code, s.category_id, s.name, s.slug, s.is_active, s.display_order, s.created_at, s.updated_at,
		        c.name AS category_name
		 FROM subcategories s
		 JOIN categories c ON s.category_id = c.id
		 WHERE s.id = $1`, id).
		Scan(&s.ID, &s.Code, &s.CategoryID, &s.Name, &s.Slug, &s.IsActive, &s.DisplayOrder,
			&s.CreatedAt, &s.UpdatedAt, &s.CategoryName)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CategoryRepository) CreateSubcategory(ctx context.Context, req models.CreateSubcategoryRequest) (*models.Subcategory, error) {
	var s models.Subcategory
	err := r.db.QueryRow(ctx,
		`INSERT INTO subcategories (category_id, name, slug, is_active, display_order)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, code, category_id, name, slug, is_active, display_order, created_at, updated_at`,
		req.CategoryID, req.Name, req.Slug, req.IsActive, req.DisplayOrder).
		Scan(&s.ID, &s.Code, &s.CategoryID, &s.Name, &s.Slug, &s.IsActive, &s.DisplayOrder,
			&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *CategoryRepository) UpdateSubcategory(ctx context.Context, id int64, req models.CreateSubcategoryRequest) (*models.Subcategory, error) {
	var s models.Subcategory
	err := r.db.QueryRow(ctx,
		`UPDATE subcategories
		 SET category_id = $2, name = $3, slug = $4, is_active = $5, display_order = $6, updated_at = NOW()
		 WHERE id = $1
		 RETURNING id, code, category_id, name, slug, is_active, display_order, created_at, updated_at`,
		id, req.CategoryID, req.Name, req.Slug, req.IsActive, req.DisplayOrder).
		Scan(&s.ID, &s.Code, &s.CategoryID, &s.Name, &s.Slug, &s.IsActive, &s.DisplayOrder,
			&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
