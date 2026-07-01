package models

import "time"

type Product struct {
	ID          string           `db:"id"`
	Code        string           `db:"code"`
	Name        string           `db:"name"`
	Brand       string           `db:"brand"`
	Model       string           `db:"model"`
	Description string           `db:"description"`
	Price       float64          `db:"price"`
	SalePrice   float64          `db:"sale_price"`  // 0 = sem promoção
	Images      []string         `db:"images"`
	Status      string           `db:"status"`
	Featured    bool             `db:"featured"`
	IsNew       bool             `db:"is_new"`
	IsSale      bool             `db:"is_sale"`
	IsLimited   bool             `db:"is_limited"`
	SortOrder   int              `db:"sort_order"`
	Slug        string           `db:"slug"`
	Variants    []ProductVariant
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`
}

// DisplayPrice retorna o preço promocional se houver, caso contrário o preço normal.
func (p *Product) DisplayPrice() float64 {
	if p.IsSale && p.SalePrice > 0 {
		return p.SalePrice
	}
	return p.Price
}

type ProductVariant struct {
	ID        string `db:"id"`
	ProductID string `db:"product_id"`
	Color     string `db:"color"`
	Size      string `db:"size"`
	Available bool   `db:"available"`
}

type ProductFilter struct {
	Search        string
	Brand         string
	Color         string
	Size          string
	MinPrice      float64
	MaxPrice      float64
	OnlyAvailable bool
	OnlyNew       bool
	OnlySale      bool
	OnlyLimited   bool
	OnlyFeatured  bool
}
