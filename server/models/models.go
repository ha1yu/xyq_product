package models

import "time"

type Category struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type Product struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	CategoryID int       `json:"category_id"`
	CategoryName string  `json:"category_name,omitempty"`
	Remark     string    `json:"remark"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Price struct {
	ID        int       `json:"id"`
	ProductID int       `json:"product_id"`
	PriceDate string    `json:"price_date"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type Admin struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type CreateProductRequest struct {
	Name       string `json:"name"`
	CategoryID int    `json:"category_id"`
	Remark     string `json:"remark"`
}

type UpdateProductRequest struct {
	Name       string `json:"name"`
	CategoryID int    `json:"category_id"`
	Remark     string `json:"remark"`
}

type CreatePriceRequest struct {
	ProductID int     `json:"product_id"`
	PriceDate string  `json:"price_date"`
	Price     float64 `json:"price"`
}

type TrendQuery struct {
	ProductIDs []string `form:"product_ids"`
}

type TrendDataPoint struct {
	Date  string  `json:"date"`
	Price float64 `json:"price"`
}

type TrendSeries struct {
	ProductID   int               `json:"product_id"`
	ProductName string            `json:"product_name"`
	Data        []TrendDataPoint  `json:"data"`
}
