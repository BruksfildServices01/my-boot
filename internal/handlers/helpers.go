package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"brl":      formatBRL,
		"colorHex": colorHex,
		"lower":    strings.ToLower,
	}
}

var colorMap = map[string]string{
	"branco":      "#F5F5F5",
	"preto":       "#111827",
	"vermelho":    "#EF4444",
	"azul":        "#3B82F6",
	"azul navy":   "#1E3A5F",
	"navy":        "#1E3A5F",
	"verde":       "#22C55E",
	"amarelo":     "#FBBF24",
	"laranja":     "#F97316",
	"rosa":        "#EC4899",
	"roxo":        "#8B5CF6",
	"cinza":       "#9CA3AF",
	"marrom":      "#92400E",
	"bege":        "#D4A76A",
	"nude":        "#E8C4A0",
	"vinho":       "#7C2D12",
	"coral":       "#F87171",
	"off white":   "#F9F5F0",
	"creme":       "#FEF9EE",
}

func colorHex(name string) string {
	// para cores compostas como "Branco/Preto", usa a primeira
	first := strings.SplitN(name, "/", 2)[0]
	first = strings.ToLower(strings.TrimSpace(first))
	if hex, ok := colorMap[first]; ok {
		return hex
	}
	// fallback: cor baseada no hash do nome
	hash := 0
	for _, c := range first {
		hash = (hash*31 + int(c)) & 0xFFFFFF
	}
	return fmt.Sprintf("#%06X", hash&0xAAAAAA|0x555555)
}

func mustParse(files ...string) *template.Template {
	return template.Must(
		template.New("").Funcs(templateFuncs()).ParseFiles(files...),
	)
}

func render(w http.ResponseWriter, tmpl *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "layout", data); err != nil {
		log.Println("template error:", err)
		http.Error(w, "Erro interno", http.StatusInternalServerError)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// formatBRL formata um float como moeda brasileira: 1349.9 → "R$ 1.349,90"
func formatBRL(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	dot := strings.LastIndex(s, ".")
	intPart, decPart := s[:dot], s[dot+1:]

	if len(intPart) > 3 {
		var b strings.Builder
		mod := len(intPart) % 3
		if mod > 0 {
			b.WriteString(intPart[:mod])
			intPart = intPart[mod:]
		}
		for len(intPart) > 0 {
			if b.Len() > 0 {
				b.WriteByte('.')
			}
			b.WriteString(intPart[:3])
			intPart = intPart[3:]
		}
		intPart = b.String()
	}

	return "R$ " + intPart + "," + decPart
}

func parsePrice(s string) (float64, error) {
	s = strings.ReplaceAll(s, "R$", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", ".")
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

func slugify(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)

	result = strings.ToLower(result)
	var b strings.Builder
	for _, r := range result {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ', r == '-', r == '_':
			b.WriteRune('-')
		}
	}

	slug := b.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}
