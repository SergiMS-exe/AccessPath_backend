package models

import "time"

type FeatureReview struct {
	ID        string    `json:"id"`
	PlaceID   string    `json:"place_id"`
	UserID    string    `json:"user_id"`
	FeatureID int32     `json:"feature_id"`
	Rating    int32     `json:"rating"`
	Comment   *string   `json:"comment,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FeatureReviewWithDetails struct {
	FeatureReview
	UserName    string `json:"user_name"`
	FeatureName string `json:"feature_name"`
}

type CreateReviewRequest struct {
	UserID    string  `json:"user_id" binding:"required"`
	FeatureID int32   `json:"feature_id" binding:"required"`
	Rating    int32   `json:"rating" binding:"required,min=1,max=5"`
	Comment   *string `json:"comment"`
}

type FeatureAverage struct {
	FeatureID   int32   `json:"feature_id"`
	FeatureName string  `json:"feature_name"`
	CategoryID  int32   `json:"category_id"`
	AverageRate float64 `json:"average_rate"`
	TotalVotes  int     `json:"total_votes"`
}
