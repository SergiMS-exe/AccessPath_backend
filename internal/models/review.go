package models

import "time"

type Review struct {
	ID        int64      `json:"id"`
	Code      string     `json:"code"`
	UserID    int64      `json:"user_id"`
	PlaceID   int64      `json:"place_id"`
	Comment   *string    `json:"comment,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type ReviewWithDetails struct {
	Review
	Username string `json:"username"`
}

// RatingInput is a single subcategory score submitted as part of a review.
type RatingInput struct {
	SubcategoryID int64 `json:"subcategory_id" binding:"required"`
	Score         int   `json:"score" binding:"required,min=1,max=5"`
}

// CreateReviewRequest is the full payload for POST /reviews.
// Photos are base64-encoded image strings; ratings are upserted atomically.
type CreateReviewRequest struct {
	PlaceID int64         `json:"place_id" binding:"required"`
	UserID  int64         `json:"user_id" binding:"required"`
	Comment *string       `json:"comment"`
	Ratings []RatingInput `json:"ratings"`
	Photos  []string      `json:"photos"` // base64-encoded image data
}
