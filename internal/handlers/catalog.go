package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"myboot/internal/models"
	"myboot/internal/repository"
)

type CatalogHandler struct {
	repo            *repository.ProductRepo
	testimonialRepo *repository.TestimonialRepo
	settingsRepo    *repository.SettingsRepo
	catalogTmpl     *template.Template
	productTmpl     *template.Template
	mybootTmpl      *template.Template
}

func NewCatalogHandler(repo *repository.ProductRepo, testimonialRepo *repository.TestimonialRepo, settingsRepo *repository.SettingsRepo) *CatalogHandler {
	return &CatalogHandler{
		repo:            repo,
		testimonialRepo: testimonialRepo,
		settingsRepo:    settingsRepo,
		catalogTmpl:     mustParse("web/templates/layout.html", "web/templates/catalog.html"),
		productTmpl:     mustParse("web/templates/layout.html", "web/templates/product.html"),
		mybootTmpl:      mustParse("web/templates/layout.html", "web/templates/myboot.html"),
	}
}

type catalogData struct {
	Products      []models.Product
	Brands        []string
	Colors        []string
	Sizes         []string
	MinPriceCat   float64
	MaxPriceCat   float64
	Filter        models.ProductFilter
	ActiveFilters int
}

func (h *CatalogHandler) Catalog(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := models.ProductFilter{
		Search:        q.Get("q"),
		Brand:         q.Get("marca"),
		Color:         q.Get("cor"),
		Size:          q.Get("tamanho"),
		OnlyAvailable: q.Get("disponiveis") == "1",
		OnlyNew:       q.Get("novos") == "1",
		OnlySale:      q.Get("promocao") == "1",
		OnlyLimited:   q.Get("limitado") == "1",
	}
	if v := q.Get("preco_min"); v != "" {
		filter.MinPrice, _ = strconv.ParseFloat(v, 64)
	}
	if v := q.Get("preco_max"); v != "" {
		filter.MaxPrice, _ = strconv.ParseFloat(v, 64)
	}

	products, err := h.repo.List(r.Context(), filter)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}

	brands, _ := h.repo.ListBrands(r.Context())
	colors, _ := h.repo.ListColors(r.Context())
	sizes, _ := h.repo.ListSizes(r.Context())
	minP, maxP := h.repo.PriceRange(r.Context())

	active := 0
	if filter.Brand != "" { active++ }
	if filter.Color != "" { active++ }
	if filter.Size != "" { active++ }
	if filter.MinPrice > 0 { active++ }
	if filter.MaxPrice > 0 { active++ }
	if filter.OnlyAvailable { active++ }
	if filter.OnlyNew { active++ }
	if filter.OnlySale { active++ }
	if filter.OnlyLimited { active++ }

	render(w, h.catalogTmpl, catalogData{
		Products:      products,
		Brands:        brands,
		Colors:        colors,
		Sizes:         sizes,
		MinPriceCat:   minP,
		MaxPriceCat:   maxP,
		Filter:        filter,
		ActiveFilters: active,
	})
}

type productData struct {
	Product        models.Product
	ColorMapJSON   template.JS
	WhatsAppNumber string
	SiteURL        string
}

func (h *CatalogHandler) MyBoot(w http.ResponseWriter, r *http.Request) {
	val, _ := h.settingsRepo.Get(r.Context(), "myboot_enabled")
	if val == "false" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	items, err := h.testimonialRepo.List(r.Context(), true)
	if err != nil {
		http.Error(w, "Erro interno", http.StatusInternalServerError)
		return
	}
	render(w, h.mybootTmpl, map[string]any{"Items": items})
}

func (h *CatalogHandler) Product(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	product, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil || product == nil {
		http.NotFound(w, r)
		return
	}

	colorMap := make(map[string][]models.ProductVariant)
	for _, v := range product.Variants {
		colorMap[v.Color] = append(colorMap[v.Color], v)
	}
	colorJSON, _ := json.Marshal(colorMap)

	render(w, h.productTmpl, productData{
		Product:        *product,
		ColorMapJSON:   template.JS(colorJSON),
		WhatsAppNumber: envOrDefault("WHATSAPP_NUMBER", ""),
		SiteURL:        envOrDefault("SITE_URL", ""),
	})
}
