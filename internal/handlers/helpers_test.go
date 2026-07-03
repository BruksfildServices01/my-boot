package handlers

import "testing"

func TestFormatBRL(t *testing.T) {
	// formatBRL usa NBSP (U+00A0) entre "R$" e o valor — convenção tipográfica brasileira.
	const nbsp = " "
	cases := []struct {
		input float64
		want  string
	}{
		{0, "R$" + nbsp + "0,00"},
		{1, "R$" + nbsp + "1,00"},
		{99.9, "R$" + nbsp + "99,90"},
		{1349.9, "R$" + nbsp + "1.349,90"},
		{12000, "R$" + nbsp + "12.000,00"},
		{1234567.89, "R$" + nbsp + "1.234.567,89"},
	}
	for _, c := range cases {
		got := formatBRL(c.input)
		if got != c.want {
			t.Errorf("formatBRL(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestParsePrice(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"R$ 1.349,90", 1349.90},
		{"99,90", 99.90},
		{"1.200,00", 1200.00},
		{"500", 500.00},
		{"R$ 12.000,00", 12000.00},
	}
	for _, c := range cases {
		got, err := parsePrice(c.input)
		if err != nil {
			t.Errorf("parsePrice(%q) error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("parsePrice(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Nike Air Max 90", "nike-air-max-90"},
		{"Tênis Adidás", "tenis-adidas"},
		// barra é removida sem gerar hífen (comportamento atual)
		{"Branco/Preto", "brancopreto"},
		{"  espaços  extras  ", "espacos-extras"},
		{"Já & Não", "ja-nao"},
	}
	for _, c := range cases {
		got := slugify(c.input)
		if got != c.want {
			t.Errorf("slugify(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
