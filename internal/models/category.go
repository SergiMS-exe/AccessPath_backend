package models

import "time"

type Category struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	IsActive     bool      `json:"is_active"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Subcategory struct {
	ID           int64     `json:"id"`
	Code         string    `json:"code"`
	CategoryID   int64     `json:"category_id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	IsActive     bool      `json:"is_active"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type SubcategoryWithCategory struct {
	Subcategory
	CategoryName string `json:"category_name"`
}

type CreateCategoryRequest struct {
	Name         string `json:"name" binding:"required"`
	Slug         string `json:"slug" binding:"required"`
	IsActive     bool   `json:"is_active"`
	DisplayOrder int    `json:"display_order"`
}

type CreateSubcategoryRequest struct {
	CategoryID   int64  `json:"category_id" binding:"required"`
	Name         string `json:"name" binding:"required"`
	Slug         string `json:"slug" binding:"required"`
	IsActive     bool   `json:"is_active"`
	DisplayOrder int    `json:"display_order"`
}
