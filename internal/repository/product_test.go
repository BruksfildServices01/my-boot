package repository

import "testing"

func TestBrandPrefix(t *testing.T) {
	cases := []struct {
		brand string
		want  string
	}{
		{"Nike", "NIK"},
		{"Adidas", "ADI"},
		{"New Balance", "NEW"},
		{"Reebok", "REE"},
		{"Puma", "PUM"},
		{"AB", "ABX"},     // menos de 3 chars → padeia com X
		{"A", "AXX"},
		{"São Paulo FC", "SAO"},
	}
	for _, c := range cases {
		got := brandPrefix(c.brand)
		if got != c.want {
			t.Errorf("brandPrefix(%q) = %q, want %q", c.brand, got, c.want)
		}
	}
}
