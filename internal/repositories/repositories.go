package repositories

import "github.com/jackc/pgx/v5/pgxpool"

// Repositories agrupa los interfaces de repositorio. Las structs concretas
// son privadas y solo se accede a ellas via las interfaces desde fuera del
// paquete, lo que permite mockearlas en tests.
type Repositories struct {
	User         UserRepository
	Place        PlaceRepository
	Catalog      CatalogRepository
	Contribution ContributionRepository
	Submission   SubmissionRepository
	Photo        PhotoRepository
	Profile      ProfileRepository
	Collection   CollectionRepository
	GmapsLog     GmapsLogRepository
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