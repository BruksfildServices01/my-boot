package models

import "testing"

func TestDisplayPrice(t *testing.T) {
	cases := []struct {
		name  string
		p     Product
		want  float64
	}{
		{"preço normal sem promoção", Product{Price: 500, SalePrice: 0, IsSale: false}, 500},
		{"em promoção com sale price", Product{Price: 500, SalePrice: 350, IsSale: true}, 350},
		{"is_sale=true mas sem sale_price", Product{Price: 500, SalePrice: 0, IsSale: true}, 500},
		{"sale price > 0 mas is_sale=false", Product{Price: 500, SalePrice: 350, IsSale: false}, 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := c.p.DisplayPrice()
			if got != c.want {
				t.Errorf("DisplayPrice() = %v, want %v", got, c.want)
			}
		})
	}
}
