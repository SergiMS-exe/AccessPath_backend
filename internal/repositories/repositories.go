package repositories

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	User         *UserRepository
	Place        *PlaceRepository
	Catalog      *CatalogRepository
	Contribution *ContributionRepository
	Submission   *SubmissionRepository
	Photo        *PhotoRepository
	Profile      *ProfileRepository
	Collection   *CollectionRepository
	GmapsLog     *GmapsLogRepository
}

func New(db *pgxpool.Pool) *Repositories {
	return &Repositories{
		User:         NewUserRepository(db),
		Place:        NewPlaceRepository(db),
		Catalog:      NewCatalogRepository(db),
		Contribution: NewContributionRepository(db),
		Submission:   NewSubmissionRepository(db),
		Photo:        NewPhotoRepository(db),
		Profile:      NewProfileRepository(db),
		Collection:   NewCollectionRepository(db),
		GmapsLog:     NewGmapsLogRepository(db),
	}
}
