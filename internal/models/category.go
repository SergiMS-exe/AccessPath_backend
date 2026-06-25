package models

import "time"

type Category struct {
	ID           int64     `db:"id" json:"id"`
	Code         string    `db:"code" json:"code"`
	Name         string    `db:"name" json:"name"`
	Slug         string    `db:"slug" json:"slug"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Subcategory struct {
	ID           int64     `db:"id" json:"id"`
	Code         string    `db:"code" json:"code"`
	CategoryID   int64     `db:"category_id" json:"category_id"`
	Name         string    `db:"name" json:"name"`
	Slug         string    `db:"slug" json:"slug"`
	IsActive     bool      `db:"is_active" json:"is_active"`
	DisplayOrder int       `db:"display_order" json:"display_order"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type SubcategoryWithCategory struct {
	Subcategory
	CategoryName string `db:"category_name" json:"category_name"`
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
