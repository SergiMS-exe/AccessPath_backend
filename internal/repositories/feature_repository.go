package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FeatureRepository struct {
	db *pgxpool.Pool
}

func NewFeatureRepository(db *pgxpool.Pool) *FeatureRepository {
	return &FeatureRepository{db: db}
}

func (r *FeatureRepository) FindAllCategories(ctx context.Context) ([]models.FeatureCategory, error) {
	query := `SELECT id, name, description, icon FROM feature_categories ORDER BY id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.FeatureCategory
	for rows.Next() {
		var c models.FeatureCategory
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.Icon); err != nil {
			return nil, err
		}
		categories = append(categories, c)
	}

	return categories, nil
}

func (r *FeatureRepository) FindAllFeatures(ctx context.Context) ([]models.FeatureWithCategory, error) {
	query := `SELECT af.id, af.category_id, af.name, af.description, af.icon, fc.name as category_name
			  FROM accessibility_features af
			  JOIN feature_categories fc ON af.category_id = fc.id
			  ORDER BY af.category_id, af.id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []models.FeatureWithCategory
	for rows.Next() {
		var f models.FeatureWithCategory
		if err := rows.Scan(&f.ID, &f.CategoryID, &f.Name, &f.Description, &f.Icon, &f.CategoryName); err != nil {
			return nil, err
		}
		features = append(features, f)
	}

	return features, nil
}

func (r *FeatureRepository) FindByCategory(ctx context.Context, categoryID int32) ([]models.AccessibilityFeature, error) {
	query := `SELECT id, category_id, name, description, icon
			  FROM accessibility_features
			  WHERE category_id = $1
			  ORDER BY id`

	rows, err := r.db.Query(ctx, query, categoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var features []models.AccessibilityFeature
	for rows.Next() {
		var f models.AccessibilityFeature
		if err := rows.Scan(&f.ID, &f.CategoryID, &f.Name, &f.Description, &f.Icon); err != nil {
			return nil, err
		}
		features = append(features, f)
	}

	return features, nil
}

func (r *FeatureRepository) FindByID(ctx context.Context, id int32) (*models.FeatureWithCategory, error) {
	query := `SELECT af.id, af.category_id, af.name, af.description, af.icon, fc.name as category_name
			  FROM accessibility_features af
			  JOIN feature_categories fc ON af.category_id = fc.id
			  WHERE af.id = $1`

	var f models.FeatureWithCategory
	err := r.db.QueryRow(ctx, query, id).Scan(&f.ID, &f.CategoryID, &f.Name, &f.Description, &f.Icon, &f.CategoryName)
	if err != nil {
		return nil, err
	}

	return &f, nil
}
