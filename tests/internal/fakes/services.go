package fakes

import (
	"context"
	"errors"

	"accesspath/internal/models"

	"accesspath/pkg/gmaps"
)

// GmapsClient es un stub de services.GmapsClient.
type GmapsClient struct {
	AutocompleteFn func(ctx context.Context, query, sessionToken string) ([]gmaps.AutocompleteItem, error)
	DetailsFn      func(ctx context.Context, placeID, sessionToken string) (*gmaps.PlaceDetails, error)
}

func (f *GmapsClient) Autocomplete(ctx context.Context, query, sessionToken string) ([]gmaps.AutocompleteItem, error) {
	if f.AutocompleteFn != nil {
		return f.AutocompleteFn(ctx, query, sessionToken)
	}
	return nil, errors.New("GmapsClient.Autocomplete not stubbed")
}

func (f *GmapsClient) Details(ctx context.Context, placeID, sessionToken string) (*gmaps.PlaceDetails, error) {
	if f.DetailsFn != nil {
		return f.DetailsFn(ctx, placeID, sessionToken)
	}
	return nil, errors.New("GmapsClient.Details not stubbed")
}

// PhotoSvc es un stub de services.PhotoServiceInterface.
type PhotoSvc struct {
	UploadFn func(ctx context.Context, data []byte) (url, objectKey string, err error)
}

func (f *PhotoSvc) Upload(ctx context.Context, data []byte) (string, string, error) {
	if f.UploadFn != nil {
		return f.UploadFn(ctx, data)
	}
	return "", "", errors.New("PhotoSvc.Upload not stubbed")
}

// SvcUser es un stub de services.UserService.
type SvcUser struct {
	GetByIDFn  func(ctx context.Context, id int64) (*models.User, error)
	RegisterFn func(ctx context.Context, req models.CreateUserRequest) (*models.User, error)
	LoginFn    func(ctx context.Context, req models.LoginRequest) (*models.User, error)
}

func (f *SvcUser) GetByID(ctx context.Context, id int64) (*models.User, error) {
	if f.GetByIDFn != nil {
		return f.GetByIDFn(ctx, id)
	}
	return nil, errors.New("SvcUser.GetByID not stubbed")
}

func (f *SvcUser) Register(ctx context.Context, req models.CreateUserRequest) (*models.User, error) {
	if f.RegisterFn != nil {
		return f.RegisterFn(ctx, req)
	}
	return nil, errors.New("SvcUser.Register not stubbed")
}

func (f *SvcUser) Login(ctx context.Context, req models.LoginRequest) (*models.User, error) {
	if f.LoginFn != nil {
		return f.LoginFn(ctx, req)
	}
	return nil, errors.New("SvcUser.Login not stubbed")
}

// SvcPlace es un stub de services.PlaceService.
type SvcPlace struct {
	GetAllFn           func(ctx context.Context, filters models.PlaceFilters) (*models.PlaceListResult, error)
	GetByBoundsFn      func(ctx context.Context, filters models.BoundsFilter) ([]models.PlaceMapItem, error)
	GetNearbyFn        func(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error)
	GetByIDFn          func(ctx context.Context, id int64) (*models.PlaceDetail, error)
	CreateFn           func(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error)
	UpdateFn           func(ctx context.Context, id, userID int64, req models.UpdatePlaceRequest) (*models.Place, error)
	DeleteFn           func(ctx context.Context, id, userID int64) error
	SearchFn           func(ctx context.Context, query, sessionToken string) ([]models.GoogleAutocompleteItem, error)
	ImportFromGoogleFn func(ctx context.Context, googlePlaceID, sessionToken string, userID int64) (*models.Place, error)
}

func (f *SvcPlace) GetAll(ctx context.Context, filters models.PlaceFilters) (*models.PlaceListResult, error) {
	if f.GetAllFn != nil {
		return f.GetAllFn(ctx, filters)
	}
	return nil, errors.New("SvcPlace.GetAll not stubbed")
}

func (f *SvcPlace) GetByBounds(ctx context.Context, filters models.BoundsFilter) ([]models.PlaceMapItem, error) {
	if f.GetByBoundsFn != nil {
		return f.GetByBoundsFn(ctx, filters)
	}
	return nil, errors.New("SvcPlace.GetByBounds not stubbed")
}

func (f *SvcPlace) GetNearby(ctx context.Context, filters models.NearbyFilter) ([]models.PlaceWithDistance, error) {
	if f.GetNearbyFn != nil {
		return f.GetNearbyFn(ctx, filters)
	}
	return nil, errors.New("SvcPlace.GetNearby not stubbed")
}

func (f *SvcPlace) GetByID(ctx context.Context, id int64) (*models.PlaceDetail, error) {
	if f.GetByIDFn != nil {
		return f.GetByIDFn(ctx, id)
	}
	return nil, errors.New("SvcPlace.GetByID not stubbed")
}

func (f *SvcPlace) Create(ctx context.Context, req models.CreatePlaceRequest) (*models.Place, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, req)
	}
	return nil, errors.New("SvcPlace.Create not stubbed")
}

func (f *SvcPlace) Update(ctx context.Context, id, userID int64, req models.UpdatePlaceRequest) (*models.Place, error) {
	if f.UpdateFn != nil {
		return f.UpdateFn(ctx, id, userID, req)
	}
	return nil, errors.New("SvcPlace.Update not stubbed")
}

func (f *SvcPlace) Delete(ctx context.Context, id, userID int64) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, id, userID)
	}
	return errors.New("SvcPlace.Delete not stubbed")
}

func (f *SvcPlace) Search(ctx context.Context, query, sessionToken string) ([]models.GoogleAutocompleteItem, error) {
	if f.SearchFn != nil {
		return f.SearchFn(ctx, query, sessionToken)
	}
	return nil, errors.New("SvcPlace.Search not stubbed")
}

func (f *SvcPlace) ImportFromGoogle(ctx context.Context, googlePlaceID, sessionToken string, userID int64) (*models.Place, error) {
	if f.ImportFromGoogleFn != nil {
		return f.ImportFromGoogleFn(ctx, googlePlaceID, sessionToken, userID)
	}
	return nil, errors.New("SvcPlace.ImportFromGoogle not stubbed")
}

// SvcSubmission es un stub de services.SubmissionService.
type SvcSubmission struct {
	SaveFn       func(ctx context.Context, userID int64, req models.SubmissionRequest) (*models.Submission, error)
	GetByPlaceFn func(ctx context.Context, placeID int64) ([]models.SubmissionWithDetails, error)
}

func (f *SvcSubmission) Save(ctx context.Context, userID int64, req models.SubmissionRequest) (*models.Submission, error) {
	if f.SaveFn != nil {
		return f.SaveFn(ctx, userID, req)
	}
	return nil, errors.New("SvcSubmission.Save not stubbed")
}

func (f *SvcSubmission) GetByPlace(ctx context.Context, placeID int64) ([]models.SubmissionWithDetails, error) {
	if f.GetByPlaceFn != nil {
		return f.GetByPlaceFn(ctx, placeID)
	}
	return nil, errors.New("SvcSubmission.GetByPlace not stubbed")
}

// SvcProfile es un stub de services.ProfileService.
type SvcProfile struct {
	GetFn    func(ctx context.Context, userID int64) (*models.ProfileResponse, error)
	SetFn    func(ctx context.Context, userID int64, req models.ProfileRequest) (*models.ProfileResponse, error)
	DeleteFn func(ctx context.Context, userID int64) error
}

func (f *SvcProfile) Get(ctx context.Context, userID int64) (*models.ProfileResponse, error) {
	if f.GetFn != nil {
		return f.GetFn(ctx, userID)
	}
	return nil, errors.New("SvcProfile.Get not stubbed")
}

func (f *SvcProfile) Set(ctx context.Context, userID int64, req models.ProfileRequest) (*models.ProfileResponse, error) {
	if f.SetFn != nil {
		return f.SetFn(ctx, userID, req)
	}
	return nil, errors.New("SvcProfile.Set not stubbed")
}

func (f *SvcProfile) Delete(ctx context.Context, userID int64) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, userID)
	}
	return errors.New("SvcProfile.Delete not stubbed")
}

// SvcCollection es un stub de services.CollectionService.
type SvcCollection struct {
	GetByUserFn   func(ctx context.Context, userID int64) ([]models.Collection, error)
	GetByIDFn     func(ctx context.Context, id int64) (*models.Collection, error)
	CreateFn      func(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error)
	DeleteFn      func(ctx context.Context, id int64) error
	AddPlaceFn    func(ctx context.Context, collectionID, placeID int64) error
	RemovePlaceFn func(ctx context.Context, collectionID, placeID int64) error
	GetPlacesFn   func(ctx context.Context, collectionID int64) ([]models.Place, error)
}

func (f *SvcCollection) GetByUser(ctx context.Context, userID int64) ([]models.Collection, error) {
	if f.GetByUserFn != nil {
		return f.GetByUserFn(ctx, userID)
	}
	return nil, errors.New("SvcCollection.GetByUser not stubbed")
}

func (f *SvcCollection) GetByID(ctx context.Context, id int64) (*models.Collection, error) {
	if f.GetByIDFn != nil {
		return f.GetByIDFn(ctx, id)
	}
	return nil, errors.New("SvcCollection.GetByID not stubbed")
}

func (f *SvcCollection) Create(ctx context.Context, req models.CreateCollectionRequest) (*models.Collection, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, req)
	}
	return nil, errors.New("SvcCollection.Create not stubbed")
}

func (f *SvcCollection) Delete(ctx context.Context, id int64) error {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, id)
	}
	return errors.New("SvcCollection.Delete not stubbed")
}

func (f *SvcCollection) AddPlace(ctx context.Context, collectionID, placeID int64) error {
	if f.AddPlaceFn != nil {
		return f.AddPlaceFn(ctx, collectionID, placeID)
	}
	return errors.New("SvcCollection.AddPlace not stubbed")
}

func (f *SvcCollection) RemovePlace(ctx context.Context, collectionID, placeID int64) error {
	if f.RemovePlaceFn != nil {
		return f.RemovePlaceFn(ctx, collectionID, placeID)
	}
	return errors.New("SvcCollection.RemovePlace not stubbed")
}

func (f *SvcCollection) GetPlaces(ctx context.Context, collectionID int64) ([]models.Place, error) {
	if f.GetPlacesFn != nil {
		return f.GetPlacesFn(ctx, collectionID)
	}
	return nil, errors.New("SvcCollection.GetPlaces not stubbed")
}

// SvcCatalog es un stub de services.CatalogService.
type SvcCatalog struct {
	GetCatalogFn func(ctx context.Context) ([]models.DimensionDetail, error)
}

func (f *SvcCatalog) GetCatalog(ctx context.Context) ([]models.DimensionDetail, error) {
	if f.GetCatalogFn != nil {
		return f.GetCatalogFn(ctx)
	}
	return nil, errors.New("SvcCatalog.GetCatalog not stubbed")
}

// SvcContribution es un stub de services.ContributionService.
type SvcContribution struct {
	CreateFn func(ctx context.Context, userID int64, req models.ContributionRequest) (*models.ContributionResult, error)
	DeleteFn func(ctx context.Context, userID, id int64) (*models.ContributionResult, error)
}

func (f *SvcContribution) Create(ctx context.Context, userID int64, req models.ContributionRequest) (*models.ContributionResult, error) {
	if f.CreateFn != nil {
		return f.CreateFn(ctx, userID, req)
	}
	return nil, errors.New("SvcContribution.Create not stubbed")
}

func (f *SvcContribution) Delete(ctx context.Context, userID, id int64) (*models.ContributionResult, error) {
	if f.DeleteFn != nil {
		return f.DeleteFn(ctx, userID, id)
	}
	return nil, errors.New("SvcContribution.Delete not stubbed")
}

// SvcQuestion es un stub de services.QuestionService.
type SvcQuestion struct {
	NextFn func(ctx context.Context, userID, placeID int64) (*models.NextQuestionResponse, error)
}

func (f *SvcQuestion) Next(ctx context.Context, userID, placeID int64) (*models.NextQuestionResponse, error) {
	if f.NextFn != nil {
		return f.NextFn(ctx, userID, placeID)
	}
	return nil, errors.New("SvcQuestion.Next not stubbed")
}
