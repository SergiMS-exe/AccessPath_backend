package models

type FeatureCategory struct {
	ID          int32   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
}

type AccessibilityFeature struct {
	ID          int32   `json:"id"`
	CategoryID  int32   `json:"category_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Icon        *string `json:"icon,omitempty"`
}

type FeatureWithCategory struct {
	AccessibilityFeature
	CategoryName string `json:"category_name"`
}
