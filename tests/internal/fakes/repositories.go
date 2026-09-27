package fakes

import (
	"context"
	"errors"
	"time"

	"accesspath/internal/models"

	"github.com/jackc/pgx/v5"
)

// RepoUser es un stub de repositories.UserRepository. Cada metodo delega en un
// campo *Fn configurable; si el campo es nil devuelve un error explicito para
// detectar tests que olvidan stubear algo.
type RepoUser struct {
	FindByIDFn    func(ctx context.Context, id int64) (*models.User, error)
	FindByCodeFn  func(ctx context.Context, code string) (*models.User, error)
	FindByEmailFn func(ctx context.Context, email string) (*models.UserWithPassword, error)
	CreateFn      func(ctx context.Context, req models.CreateUserRequest, hashedPassword string) (*models.User, error)
	DeleteFn      func(ctx context.Context, id int64) error
}

func (f *RepoUser) FindByID(ctx context.Context, id int64) (*models.User, error) {
	if f.FindByIDFn != nil {
		return f.FindByIDFn(ctx, id)
	}
	return nil, errors.New("RepoUser.FindByID not stubbed")
}

func (f *RepoUser) FindByCode(ctx context.Context, code string) (*models.User, error) {
	if f.FindByCodeFn != nil {
		return f.FindByCodeFn(ctx, code)
	}
	return nil, errors.New("RepoUser.FindByCode not stubbed")
}

func (f *RepoUser) FindByEmail(ctx context.Context, email string) (*models.UserWithPassword, error) {
	if f.FindByEmailFn != nil {
		return f.FindByEmailFn(ctx, email)
	}
	return nil, errors.New("RepoUser.FindByEmail not stubbed")
}

func (f *RepoUser) Create(ctx context.Context, req models.CreateUserRequest, hashedPassword string) (*models.User, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, req, hashedPassword)
	}
	return nil, errors.New("RepoUser.Create not stubbed")
}

func (f *RepoUser) Delete(ctx context.Context, id int64) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, id)
	}
	return errors.New("RepoUser.Delete not stubbed")
}

// RepoPlace es un stub de repositories.PlaceRepository.
type RepoPlace struct {
	FindAllFn              func(ctx context.Context, filters models.PlaceFilters) ([]models.Place, int, error)
	FindByIDFn             func(ctx context.Context, id int64) (*models.Place, error)
	FindByCodeFn           func(ctx context.Context, code string) (*models.Place, error)
	FindByBoundsFn         func(ctx context.Context, f models.BoundsFilter) ([]models.Place, error)
	FindNearbyFn           func(ctx context.Context, f models.NearbyFilter) ([]models.PlaceWithDistance, error)
	CreateFn               func(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error)
	FindByGooglePlaceIDFn  func(ctx context.Context, googlePlaceID string) (*models.Place, error)
	MarkPublishedTxFn      func(ctx context.Context, tx pgx.Tx, placeID int64) error
	UpdateFn               func(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error)
	DeleteFn               func(ctx context.Context, id int64) error
	GetAccessibilityRowsFn func(ctx context.Context, placeID int64) ([]models.CriterionAggRow, error)
	GetCriterionAggRowFn   func(ctx context.Context, placeID, criterionID int64) (*models.CriterionAggRow, error)
	GetAggRowsByPlaceIDsFn func(ctx context.Context, placeIDs []int64) ([]models.CriterionAggRow, error)
}

func (f *RepoPlace) FindAll(ctx context.Context, filters models.PlaceFilters) ([]models.Place, int, error) {
	if f.FindAllFn != nil {
		return f.FindAllFn(ctx, filters)
	}
	return nil, 0, errors.New("RepoPlace.FindAll not stubbed")
}

func (f *RepoPlace) FindByID(ctx context.Context, id int64) (*models.Place, error) {
	if f.FindByIDFn != nil {
		return f.FindByIDFn(ctx, id)
	}
	return nil, errors.New("RepoPlace.FindByID not stubbed")
}

func (f *RepoPlace) FindByCode(ctx context.Context, code string) (*models.Place, error) {
	if f.FindByCodeFn != nil {
		return f.FindByCodeFn(ctx, code)
	}
	return nil, errors.New("RepoPlace.FindByCode not stubbed")
}

func (f *RepoPlace) FindByBounds(ctx context.Context, fl models.BoundsFilter) ([]models.Place, error) {
	if f.FindByBoundsFn != nil {
		return f.FindByBoundsFn(ctx, fl)
	}
	return nil, errors.New("RepoPlace.FindByBounds not stubbed")
}

func (f *RepoPlace) FindNearby(ctx context.Context, fl models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	if f.FindNearbyFn != nil {
		return f.FindNearbyFn(ctx, fl)
	}
	return nil, errors.New("RepoPlace.FindNearby not stubbed")
}

func (f *RepoPlace) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, req)
	}
	return nil, errors.New("RepoPlace.Create not stubbed")
}

func (f *RepoPlace) FindByGooglePlaceID(ctx context.Context, googlePlaceID string) (*models.Place, error) {
	if f.FindByGooglePlaceIDFn != nil {
		return f.FindByGooglePlaceIDFn(ctx, googlePlaceID)
	}
	return nil, errors.New("RepoPlace.FindByGooglePlaceID not stubbed")
}

func (f *RepoPlace) MarkPublishedTx(ctx context.Context, tx pgx.Tx, placeID int64) error {
	if f.MarkPublishedTxFn != nil {
		return f.MarkPublishedTxFn(ctx, tx, placeID)
	}
	return errors.New("RepoPlace.MarkPublishedTx not stubbed")
}

func (f *RepoPlace) Update(ctx context.Context, id int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	if f.UpdateFn != nil {
		return f.UpdateFn(ctx, id, req)
	}
	return nil, errors.New("RepoPlace.Update not stubbed")
}

func (f *RepoPlace) Delete(ctx context.Context, id int64) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, id)
	}
	return errors.New("RepoPlace.Delete not stubbed")
}

func (f *RepoPlace) GetAccessibilityRows(ctx context.Context, placeID int64) ([]models.CriterionAggRow, error) {
	if f.GetAccessibilityRowsFn != nil {
		return f.GetAccessibilityRowsFn(ctx, placeID)
	}
	return nil, errors.New("RepoPlace.GetAccessibilityRows not stubbed")
}

func (f *RepoPlace) GetCriterionAggRow(ctx context.Context, placeID, criterionID int64) (*models.CriterionAggRow, error) {
	if f.GetCriterionAggRowFn != nil {
		return f.GetCriterionAggRowFn(ctx, placeID, criterionID)
	}
	return nil, errors.New("RepoPlace.GetCriterionAggRow not stubbed")
}

func (f *RepoPlace) GetAggRowsByPlaceIDs(ctx context.Context, placeIDs []int64) ([]models.CriterionAggRow, error) {
	if f.GetAggRowsByPlaceIDsFn != nil {
		return f.GetAggRowsByPlaceIDsFn(ctx, placeIDs)
	}
	return nil, errors.New("RepoPlace.GetAggRowsByPlaceIDs not stubbed")
}

// RepoSubmission es un stub de repositories.SubmissionRepository.
type RepoSubmission struct {
	GetOrCreateLiveTxFn func(ctx context.Context, tx pgx.Tx, userID, placeID int64) (*models.Submission, error)
	SetCommentTxFn      func(ctx context.Context, tx pgx.Tx, submissionID int64, comment *string) error
	FindByPlaceFn       func(ctx context.Context, placeID int64) ([]models.SubmissionWithDetails, error)
}

func (f *RepoSubmission) GetOrCreateLiveTx(ctx context.Context, tx pgx.Tx, userID, placeID int64) (*models.Submission, error) {
	if f.GetOrCreateLiveTxFn != nil {
		return f.GetOrCreateLiveTxFn(ctx, tx, userID, placeID)
	}
	return nil, errors.New("RepoSubmission.GetOrCreateLiveTx not stubbed")
}

func (f *RepoSubmission) SetCommentTx(ctx context.Context, tx pgx.Tx, submissionID int64, comment *string) error {
	if f.SetCommentTxFn != nil {
		return f.SetCommentTxFn(ctx, tx, submissionID, comment)
	}
	return errors.New("RepoSubmission.SetCommentTx not stubbed")
}

func (f *RepoSubmission) FindByPlace(ctx context.Context, placeID int64) ([]models.SubmissionWithDetails, error) {
	if f.FindByPlaceFn != nil {
		return f.FindByPlaceFn(ctx, placeID)
	}
	return nil, errors.New("RepoSubmission.FindByPlace not stubbed")
}

// RepoPhoto es un stub de repositories.PhotoRepository.
type RepoPhoto struct {
	SaveTxFn              func(ctx context.Context, tx pgx.Tx, submissionID int64, contributionID *int64, url string, objectKey, suggestedSlot *string) (*models.Photo, error)
	FindBySubmissionIDsFn func(ctx context.Context, submissionIDs []int64) ([]models.Photo, error)
}

func (f *RepoPhoto) SaveTx(ctx context.Context, tx pgx.Tx, submissionID int64, contributionID *int64, url string, objectKey, suggestedSlot *string) (*models.Photo, error) {
	if f.SaveTxFn != nil {
		return f.SaveTxFn(ctx, tx, submissionID, contributionID, url, objectKey, suggestedSlot)
	}
	return nil, errors.New("RepoPhoto.SaveTx not stubbed")
}

func (f *RepoPhoto) FindBySubmissionIDs(ctx context.Context, submissionIDs []int64) ([]models.Photo, error) {
	if f.FindBySubmissionIDsFn != nil {
		return f.FindBySubmissionIDsFn(ctx, submissionIDs)
	}
	return nil, errors.New("RepoPhoto.FindBySubmissionIDs not stubbed")
}

// RepoCollection es un stub de repositories.CollectionRepository.
type RepoCollection struct {
	FindByUserFn  func(ctx context.Context, userID int64) ([]models.Collection, error)
	FindByIDFn    func(ctx context.Context, id int64) (*models.Collection, error)
	CreateFn      func(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error)
	DeleteFn      func(ctx context.Context, id int64) error
	AddPlaceFn    func(ctx context.Context, collectionID, placeID int64) error
	RemovePlaceFn func(ctx context.Context, collectionID, placeID int64) error
	GetPlacesFn   func(ctx context.Context, collectionID int64) ([]models.Place, error)
}

func (f *RepoCollection) FindByUser(ctx context.Context, userID int64) ([]models.Collection, error) {
	if f.FindByUserFn != nil {
		return f.FindByUserFn(ctx, userID)
	}
	return nil, errors.New("RepoCollection.FindByUser not stubbed")
}

func (f *RepoCollection) FindByID(ctx context.Context, id int64) (*models.Collection, error) {
	if f.FindByIDFn != nil {
		return f.FindByIDFn(ctx, id)
	}
	return nil, errors.New("RepoCollection.FindByID not stubbed")
}

func (f *RepoCollection) Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, req)
	}
	return nil, errors.New("RepoCollection.Create not stubbed")
}

func (f *RepoCollection) Delete(ctx context.Context, id int64) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, id)
	}
	return errors.New("RepoCollection.Delete not stubbed")
}

func (f *RepoCollection) AddPlace(ctx context.Context, collectionID, placeID int64) error {
	if f.AddPlaceFn != nil {
		return f.AddPlaceFn(ctx, collectionID, placeID)
	}
	return errors.New("RepoCollection.AddPlace not stubbed")
}

func (f *RepoCollection) RemovePlace(ctx context.Context, collectionID, placeID int64) error {
	if f.RemovePlaceFn != nil {
		return f.RemovePlaceFn(ctx, collectionID, placeID)
	}
	return errors.New("RepoCollection.RemovePlace not stubbed")
}

func (f *RepoCollection) GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error) {
	if f.GetPlacesFn != nil {
		return f.GetPlacesFn(ctx, collectionID)
	}
	return nil, errors.New("RepoCollection.GetPlaces not stubbed")
}

// RepoCatalog es un stub de repositories.CatalogRepository.
type RepoCatalog struct {
	GetCatalogFn            func(ctx context.Context) ([]models.DimensionDetail, error)
	GetOptionByIDFn         func(ctx context.Context, id int64) (*models.AnswerOption, error)
	GetOptionsByCriterionFn func(ctx context.Context, criterionID int64) ([]models.AnswerOption, error)
	GetCriterionByIDFn      func(ctx context.Context, id int64) (*models.Criterion, error)
}

func (f *RepoCatalog) GetCatalog(ctx context.Context) ([]models.DimensionDetail, error) {
	if f.GetCatalogFn != nil {
		return f.GetCatalogFn(ctx)
	}
	return nil, errors.New("RepoCatalog.GetCatalog not stubbed")
}

func (f *RepoCatalog) GetOptionByID(ctx context.Context, id int64) (*models.AnswerOption, error) {
	if f.GetOptionByIDFn != nil {
		return f.GetOptionByIDFn(ctx, id)
	}
	return nil, errors.New("RepoCatalog.GetOptionByID not stubbed")
}

func (f *RepoCatalog) GetOptionsByCriterion(ctx context.Context, criterionID int64) ([]models.AnswerOption, error) {
	if f.GetOptionsByCriterionFn != nil {
		return f.GetOptionsByCriterionFn(ctx, criterionID)
	}
	return nil, errors.New("RepoCatalog.GetOptionsByCriterion not stubbed")
}

func (f *RepoCatalog) GetCriterionByID(ctx context.Context, id int64) (*models.Criterion, error) {
	if f.GetCriterionByIDFn != nil {
		return f.GetCriterionByIDFn(ctx, id)
	}
	return nil, errors.New("RepoCatalog.GetCriterionByID not stubbed")
}

// RepoContribution es un stub de repositories.ContributionRepository.
type RepoContribution struct {
	UpsertLiveTxFn       func(ctx context.Context, tx pgx.Tx, submissionID, userID, placeID, criterionID, answerOptionID int64, existsFlag *bool, quality *int) (*models.Contribution, error)
	FindByIDFn           func(ctx context.Context, id int64) (*models.Contribution, error)
	SoftDeleteTxFn       func(ctx context.Context, tx pgx.Tx, id int64) error
	AnsweredByPlaceFn    func(ctx context.Context, userID, placeID int64) ([]models.AnsweredContribution, error)
	RecalculateCacheTxFn func(ctx context.Context, tx pgx.Tx, placeID, criterionID int64) error
}

func (f *RepoContribution) UpsertLiveTx(ctx context.Context, tx pgx.Tx, submissionID, userID, placeID, criterionID, answerOptionID int64, existsFlag *bool, quality *int) (*models.Contribution, error) {
	if f.UpsertLiveTxFn != nil {
		return f.UpsertLiveTxFn(ctx, tx, submissionID, userID, placeID, criterionID, answerOptionID, existsFlag, quality)
	}
	return nil, errors.New("RepoContribution.UpsertLiveTx not stubbed")
}

func (f *RepoContribution) FindByID(ctx context.Context, id int64) (*models.Contribution, error) {
	if f.FindByIDFn != nil {
		return f.FindByIDFn(ctx, id)
	}
	return nil, errors.New("RepoContribution.FindByID not stubbed")
}

func (f *RepoContribution) SoftDeleteTx(ctx context.Context, tx pgx.Tx, id int64) error {
	if f.SoftDeleteTxFn != nil {
		return f.SoftDeleteTxFn(ctx, tx, id)
	}
	return errors.New("RepoContribution.SoftDeleteTx not stubbed")
}

func (f *RepoContribution) AnsweredByPlace(ctx context.Context, userID, placeID int64) ([]models.AnsweredContribution, error) {
	if f.AnsweredByPlaceFn != nil {
		return f.AnsweredByPlaceFn(ctx, userID, placeID)
	}
	return nil, errors.New("RepoContribution.AnsweredByPlace not stubbed")
}

func (f *RepoContribution) RecalculateCacheTx(ctx context.Context, tx pgx.Tx, placeID, criterionID int64) error {
	if f.RecalculateCacheTxFn != nil {
		return f.RecalculateCacheTxFn(ctx, tx, placeID, criterionID)
	}
	return errors.New("RepoContribution.RecalculateCacheTx not stubbed")
}

// RepoProfile es un stub de repositories.ProfileRepository.
type RepoProfile struct {
	GetNeedsFn       func(ctx context.Context, userID int64) ([]string, error)
	GetConsentFn     func(ctx context.Context, userID int64) (*time.Time, error)
	ReplaceNeedsTxFn func(ctx context.Context, tx pgx.Tx, userID int64, needs []string) error
	SetConsentTxFn   func(ctx context.Context, tx pgx.Tx, userID int64) error
	ClearConsentTxFn func(ctx context.Context, tx pgx.Tx, userID int64) error
}

func (f *RepoProfile) GetNeeds(ctx context.Context, userID int64) ([]string, error) {
	if f.GetNeedsFn != nil {
		return f.GetNeedsFn(ctx, userID)
	}
	return nil, errors.New("RepoProfile.GetNeeds not stubbed")
}

func (f *RepoProfile) GetConsent(ctx context.Context, userID int64) (*time.Time, error) {
	if f.GetConsentFn != nil {
		return f.GetConsentFn(ctx, userID)
	}
	return nil, errors.New("RepoProfile.GetConsent not stubbed")
}

func (f *RepoProfile) ReplaceNeedsTx(ctx context.Context, tx pgx.Tx, userID int64, needs []string) error {
	if f.ReplaceNeedsTxFn != nil {
		return f.ReplaceNeedsTxFn(ctx, tx, userID, needs)
	}
	return errors.New("RepoProfile.ReplaceNeedsTx not stubbed")
}

func (f *RepoProfile) SetConsentTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	if f.SetConsentTxFn != nil {
		return f.SetConsentTxFn(ctx, tx, userID)
	}
	return errors.New("RepoProfile.SetConsentTx not stubbed")
}

func (f *RepoProfile) ClearConsentTx(ctx context.Context, tx pgx.Tx, userID int64) error {
	if f.ClearConsentTxFn != nil {
		return f.ClearConsentTxFn(ctx, tx, userID)
	}
	return errors.New("RepoProfile.ClearConsentTx not stubbed")
}

// RepoGmapsLog es un stub de repositories.GmapsLogRepository.
type RepoGmapsLog struct {
	CountThisMonthFn func(ctx context.Context) (int, error)
	LogFn            func(ctx context.Context) error
}

func (f *RepoGmapsLog) CountThisMonth(ctx context.Context) (int, error) {
	if f.CountThisMonthFn != nil {
		return f.CountThisMonthFn(ctx)
	}
	return 0, errors.New("RepoGmapsLog.CountThisMonth not stubbed")
}

func (f *RepoGmapsLog) Log(ctx context.Context) error {
	if f.LogFn != nil {
		return f.LogFn(ctx)
	}
	return errors.New("RepoGmapsLog.Log not stubbed")
}
