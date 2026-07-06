package services

import (
	"context"
	"fmt"
	"math/rand"
	"sort"

	"accesspath/internal/config"
	"accesspath/internal/models"
	"accesspath/internal/repositories"
)

type QuestionService struct {
	cfg         *config.AccessibilityThresholds
	catalogRepo *repositories.CatalogRepository
	placeRepo   *repositories.PlaceRepository
	contribRepo *repositories.ContributionRepository
	profileRepo *repositories.ProfileRepository
	accSvc      *AccessibilityService
}

func NewQuestionService(
	cfg *config.AccessibilityThresholds,
	catalogRepo *repositories.CatalogRepository,
	placeRepo *repositories.PlaceRepository,
	contribRepo *repositories.ContributionRepository,
	profileRepo *repositories.ProfileRepository,
	accSvc *AccessibilityService,
) *QuestionService {
	return &QuestionService{
		cfg:         cfg,
		catalogRepo: catalogRepo,
		placeRepo:   placeRepo,
		contribRepo: contribRepo,
		profileRepo: profileRepo,
		accSvc:      accSvc,
	}
}

type candidate struct {
	criterion models.Criterion
	score     float64
	isStarter bool
}

// Next elige el criterio de mayor valor para (place, user), respetando
// depends_on, perfil, exclusion de respondidos y prioridad de gris/conflicto.
// No hay decaimiento ni re-pregunta por antiguedad. Devuelve una respuesta
// vacia si no hay nada util que preguntar.
func (s *QuestionService) Next(ctx context.Context, userID, placeID int64) (*models.NextQuestionResponse, error) {
	catalog, err := s.catalogRepo.GetCatalog(ctx)
	if err != nil {
		return nil, fmt.Errorf("question: catalog: %w", err)
	}
	aggRowsList, err := s.placeRepo.GetAccessibilityRows(ctx, placeID)
	if err != nil {
		return nil, fmt.Errorf("question: agg rows: %w", err)
	}
	answeredList, err := s.contribRepo.AnsweredByPlace(ctx, userID, placeID)
	if err != nil {
		return nil, fmt.Errorf("question: answered: %w", err)
	}
	needsList, err := s.profileRepo.GetNeeds(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("question: needs: %w", err)
	}

	aggByCrit := map[int64]models.CriterionAggRow{}
	for _, row := range aggRowsList {
		aggByCrit[row.CriterionID] = row
	}
	answered := map[int64]models.AnsweredContribution{}
	for _, a := range answeredList {
		answered[a.CriterionID] = a
	}
	needs := map[string]bool{}
	for _, n := range needsList {
		needs[n] = true
	}
	optsByCrit := map[int64][]models.AnswerOption{}
	stateByCrit := map[int64]models.AccessibilityState{}
	for _, dim := range catalog {
		for _, cd := range dim.Criteria {
			optsByCrit[cd.ID] = cd.Options
			stateByCrit[cd.ID] = s.accSvc.DeriveCriterion(aggByCrit[cd.ID]).State
		}
	}

	isFirst := len(answered) == 0

	var candidates []candidate
	for _, dim := range catalog {
		for _, cd := range dim.Criteria {
			crit := cd.Criterion

			// Excluir criterios ya respondidos por el usuario. No hay re-pregunta
			// por antiguedad: la recencia nunca es un mecanismo del sistema.
			if _, ok := answered[crit.ID]; ok {
				continue
			}

			// Excluir criterios con depends_on no satisfecho.
			if crit.DependsOnCriterionID != nil {
				if !s.existsPositive(*crit.DependsOnCriterionID, answered, stateByCrit) {
					continue
				}
			}

			row := aggByCrit[crit.ID]
			defined := row.NYes + row.NNo

			valor := 1.0 / float64(1+defined)
			if stateByCrit[crit.ID] == models.StateYellow && s.accSvc.DeriveCriterion(row).Conflict {
				valor += s.cfg.Selection.ConflictBonus
			}

			relevancia := s.cfg.Selection.RelevanceMatch
			if len(needs) > 0 && !intersects(crit.ProfileTags, needs) {
				relevancia = s.cfg.Selection.RelevanceNoMatch
			}

			candidates = append(candidates, candidate{
				criterion: crit,
				score:     valor * relevancia * float64(crit.Weight),
				isStarter: crit.IsStarter,
			})
		}
	}

	// Primera pregunta de la sesion: preferir starters si los hay.
	if isFirst {
		var starters []candidate
		for _, c := range candidates {
			if c.isStarter {
				starters = append(starters, c)
			}
		}
		if len(starters) > 0 {
			candidates = starters
		}
	}

	if len(candidates) == 0 {
		return &models.NextQuestionResponse{}, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	topK := s.cfg.Selection.TopK
	if topK <= 0 || topK > len(candidates) {
		topK = len(candidates)
	}
	chosen := candidates[rand.Intn(topK)].criterion

	return &models.NextQuestionResponse{
		Criterion: &chosen,
		Options:   optsByCrit[chosen.ID],
	}, nil
}

// existsPositive indica si el criterio del que se depende tiene exists=true, ya
// sea por respuesta propia del usuario o por consenso del lugar (estado no rojo
// ni gris).
func (s *QuestionService) existsPositive(depID int64, answered map[int64]models.AnsweredContribution, stateByCrit map[int64]models.AccessibilityState) bool {
	if a, ok := answered[depID]; ok && a.ExistsFlag != nil && *a.ExistsFlag {
		return true
	}
	st := stateByCrit[depID]
	return st == models.StateGreen || st == models.StateYellow
}

func intersects(tags []string, needs map[string]bool) bool {
	for _, t := range tags {
		if needs[t] {
			return true
		}
	}
	return false
}
