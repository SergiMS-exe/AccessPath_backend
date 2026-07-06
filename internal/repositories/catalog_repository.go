package repositories

import (
	"context"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogRepository lee el catalogo del formulario: dimension -> criterion -> option.
type CatalogRepository struct {
	db *pgxpool.Pool
}

func NewCatalogRepository(db *pgxpool.Pool) *CatalogRepository {
	return &CatalogRepository{db: db}
}

const criterionColumns = `id, code, dimension_id, key, prompt, profile_tags, weight, is_blocking, is_starter, depends_on_criterion_id, sort_order, active`
const answerOptionColumns = `id, criterion_id, label, exists_value, quality_value, is_unsure, sort_order`

// GetCatalog devuelve las dimensiones activas con sus criterios activos y opciones,
// ordenado por sort_order. Ensambla el arbol en Go a partir de tres queries.
func (r *CatalogRepository) GetCatalog(ctx context.Context) ([]models.DimensionDetail, error) {
	dimRows, err := r.db.Query(ctx,
		`SELECT id, code, key, name, description, sort_order, active
		 FROM dimension WHERE active = TRUE ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	dimensions, err := pgx.CollectRows(dimRows, pgx.RowToStructByName[models.Dimension])
	if err != nil {
		return nil, err
	}

	critRows, err := r.db.Query(ctx,
		`SELECT `+criterionColumns+`
		 FROM criterion WHERE active = TRUE ORDER BY dimension_id, sort_order, id`)
	if err != nil {
		return nil, err
	}
	criteria, err := pgx.CollectRows(critRows, pgx.RowToStructByName[models.Criterion])
	if err != nil {
		return nil, err
	}

	optRows, err := r.db.Query(ctx,
		`SELECT `+answerOptionColumns+`
		 FROM answer_option ORDER BY criterion_id, sort_order, id`)
	if err != nil {
		return nil, err
	}
	options, err := pgx.CollectRows(optRows, pgx.RowToStructByName[models.AnswerOption])
	if err != nil {
		return nil, err
	}

	optsByCrit := map[int64][]models.AnswerOption{}
	for _, o := range options {
		optsByCrit[o.CriterionID] = append(optsByCrit[o.CriterionID], o)
	}

	critsByDim := map[int64][]models.CriterionDetail{}
	for _, c := range criteria {
		critsByDim[c.DimensionID] = append(critsByDim[c.DimensionID], models.CriterionDetail{
			Criterion: c,
			Options:   optsByCrit[c.ID],
		})
	}

	result := make([]models.DimensionDetail, 0, len(dimensions))
	for _, d := range dimensions {
		result = append(result, models.DimensionDetail{
			Dimension: d,
			Criteria:  critsByDim[d.ID],
		})
	}
	return result, nil
}

// GetOptionByID resuelve una opcion (para copiar exists/quality al crear contribucion).
func (r *CatalogRepository) GetOptionByID(ctx context.Context, id int64) (*models.AnswerOption, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+answerOptionColumns+` FROM answer_option WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	opt, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.AnswerOption])
	if err != nil {
		return nil, err
	}
	return &opt, nil
}

// GetOptionsByCriterion devuelve las opciones de un criterio (para next-question).
func (r *CatalogRepository) GetOptionsByCriterion(ctx context.Context, criterionID int64) ([]models.AnswerOption, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+answerOptionColumns+`
		 FROM answer_option WHERE criterion_id = $1 ORDER BY sort_order, id`, criterionID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[models.AnswerOption])
}

// GetCriterionByID resuelve un criterio (metadata para el semaforo en vivo).
func (r *CatalogRepository) GetCriterionByID(ctx context.Context, id int64) (*models.Criterion, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+criterionColumns+` FROM criterion WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	crit, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[models.Criterion])
	if err != nil {
		return nil, err
	}
	return &crit, nil
}
