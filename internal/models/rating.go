package models

type SubcategoryRating struct {
	SubcategoryID   int     `json:"subcategory_id"`
	SubcategoryName string  `json:"subcategory_name"`
	AvgScore        float64 `json:"avg_score"`
	TotalRatings    int     `json:"total_ratings"`
}

type CategoryRating struct {
	CategoryID    int                 `json:"category_id"`
	CategoryName  string              `json:"category_name"`
	AvgScore      float64             `json:"avg_score"`
	Subcategories []SubcategoryRating `json:"subcategories"`
}
