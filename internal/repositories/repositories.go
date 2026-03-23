package repositories

import "github.com/jackc/pgx/v5/pgxpool"

type Repositories struct {
	User       *UserRepository
	Place      *PlaceRepository
	Review     *ReviewRepository
	Category   *CategoryRepository
	Collection *CollectionRepository
	Rating     *RatingRepository
	Photo      *PhotoRepository
}

func New(db *pgxpool.Pool) *Repositories {
	return &Repositories{
		User:       NewUserRepository(db),
		Place:      NewPlaceRepository(db),
		Review:     NewReviewRepository(db),
		Category:   NewCategoryRepository(db),
		Collection: NewCollectionRepository(db),
		Rating:     NewRatingRepository(db),
		Photo:      NewPhotoRepository(db),
	}
}
