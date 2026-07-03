package handlers

import (
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"myboot/internal/models"
	"myboot/internal/repository"
	"myboot/internal/session"
)

// loginLimiter bloqueia IPs após 5 tentativas falhas por 15 minutos.
type loginLimiter struct {
	mu      sync.Mutex
	entries map[string]*limitEntry
}

type limitEntry struct {
	failures int
	blockedUntil time.Time
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{entries: make(map[string]*limitEntry)}
}

func (l *loginLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[ip]
	if e == nil {
		return true
	}
	if !e.blockedUntil.IsZero() && time.Now().Before(e.blockedUntil) {
		return false
	}
	return true
}

func (l *loginLimiter) recordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e := l.entries[ip]
	if e == nil {
		e = &limitEntry{}
		l.entries[ip] = e
	}
	// reseta contador se o bloqueio anterior já expirou
	if !e.blockedUntil.IsZero() && time.Now().After(e.blockedUntil) {
		e.failures = 0
		e.blockedUntil = time.Time{}
	}
	e.failures++
	if e.failures >= 5 {
		e.blockedUntil = time.Now().Add(15 * time.Minute)
	}
}

func (l *loginLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.SplitN(fwd, ",", 2)[0])
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

type AdminHandler struct {
	repo            *repository.ProductRepo
	testimonialRepo *repository.TestimonialRepo
	settingsRepo    *repository.SettingsRepo
	sessions        *session.Store
	upload          *UploadHandler
	limiter         *loginLimiter

	loginTmpl           *template.Template
	listTmpl            *template.Template
	formTmpl            *template.Template
	testimonialsTmpl    *template.Template
	testimonialFormTmpl *template.Template
}

func NewAdminHandler(repo *repository.ProductRepo, testimonialRepo *repository.TestimonialRepo, settingsRepo *repository.SettingsRepo, sessions *session.Store, upload *UploadHandler) *AdminHandler {
	return &AdminHandler{
		repo:                repo,
		testimonialRepo:     testimonialRepo,
		settingsRepo:        settingsRepo,
		sessions:            sessions,
		upload:              upload,
		limiter:             newLoginLimiter(),
		loginTmpl:           mustParse("web/templates/admin/layout.html", "web/templates/admin/login.html"),
		listTmpl:            mustParse("web/templates/admin/layout.html", "web/templates/admin/products.html"),
		formTmpl:            mustParse("web/templates/admin/layout.html", "web/templates/admin/product_form.html"),
		testimonialsTmpl:    mustParse("web/templates/admin/layout.html", "web/templates/admin/testimonials.html"),
		testimonialFormTmpl: mustParse("web/templates/admin/layout.html", "web/templates/admin/testimonial_form.html"),
	}
}

// AuthMiddleware bloqueia rotas do admin sem sessão válida.
func (h *AdminHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !h.sessions.IsValid(r) {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Auth ---

func (h *AdminHandler) Login(w http.ResponseWriter, r *http.Request) {
	render(w, h.loginTmpl, map[string]string{"Error": ""})
}

func (h *AdminHandler) LoginPost(w http.ResponseWriter, r *http.Request) {
	ip := clientIP(r)

	if !h.limiter.allow(ip) {
		render(w, h.loginTmpl, map[string]string{"Error": "Muitas tentativas. Tente novamente em 15 minutos."})
		return
	}

	user := r.FormValue("usuario")
	pass := r.FormValue("senha")

	if user == os.Getenv("ADMIN_USER") && pass == os.Getenv("ADMIN_PASSWORD") {
		h.limiter.reset(ip)
		h.sessions.Create(w)
		http.Redirect(w, r, "/admin/produtos", http.StatusFound)
		return
	}

	h.limiter.recordFailure(ip)
	render(w, h.loginTmpl, map[string]string{"Error": "Usuário ou senha inválidos."})
}

func (h *AdminHandler) Logout(w http.ResponseWriter, r *http.Request) {
	h.sessions.Destroy(w, r)
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

// --- Produtos ---

type adminStats struct {
	Total       int
	Available   int
	Unavailable int
	OnSale      int
	Featured    int
	IsNew       int
	IsLimited   int
}

type adminListData struct {
	Products []models.Product
	Brands   []string
	Stats    adminStats
	Flash    string
}

func (h *AdminHandler) Products(w http.ResponseWriter, r *http.Request) {
	products, err := h.repo.List(r.Context(), models.ProductFilter{})
	if err != nil {
		http.Error(w, "Erro ao carregar produtos", http.StatusInternalServerError)
		return
	}

	brands, _ := h.repo.ListBrands(r.Context())

	var stats adminStats
	for _, p := range products {
		stats.Total++
		if p.Status == "available" {
			stats.Available++
		} else {
			stats.Unavailable++
		}
		if p.IsSale    { stats.OnSale++ }
		if p.Featured  { stats.Featured++ }
		if p.IsNew     { stats.IsNew++ }
		if p.IsLimited { stats.IsLimited++ }
	}

	render(w, h.listTmpl, adminListData{
		Products: products,
		Brands:   brands,
		Stats:    stats,
		Flash:    r.URL.Query().Get("flash"),
	})
}

type adminFormData struct {
	Product        *models.Product
	VariantsText   string
	IsEdit         bool
	Error          string
}

func (h *AdminHandler) NewProduct(w http.ResponseWriter, r *http.Request) {
	render(w, h.formTmpl, adminFormData{Product: &models.Product{Status: "available"}})
}

func (h *AdminHandler) EditProduct(w http.ResponseWriter, r *http.Request) {
	product, err := h.repo.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil || product == nil {
		http.NotFound(w, r)
		return
	}
	render(w, h.formTmpl, adminFormData{
		Product:      product,
		VariantsText: variantsToText(product.Variants),
		IsEdit:       true,
	})
}

func (h *AdminHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	product, errMsg := h.parseForm(r)
	if errMsg != "" {
		render(w, h.formTmpl, adminFormData{Product: product, VariantsText: r.FormValue("variantes"), Error: errMsg})
		return
	}

	if product.Code == "" {
		code, err := h.repo.GenerateCode(r.Context(), product.Brand)
		if err != nil || code == "" {
			code = slugify(product.Name)[:6]
		}
		product.Code = strings.ToUpper(code)
	}

	if err := h.repo.Create(r.Context(), product); err != nil {
		render(w, h.formTmpl, adminFormData{Product: product, VariantsText: r.FormValue("variantes"), Error: "Erro ao salvar: " + err.Error()})
		return
	}

	http.Redirect(w, r, "/admin/produtos?flash=Produto+criado+com+sucesso", http.StatusFound)
}

func (h *AdminHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	product, errMsg := h.parseForm(r)
	if errMsg != "" {
		render(w, h.formTmpl, adminFormData{Product: product, VariantsText: r.FormValue("variantes"), IsEdit: true, Error: errMsg})
		return
	}
	product.ID = chi.URLParam(r, "id")

	if err := h.repo.Update(r.Context(), product); err != nil {
		render(w, h.formTmpl, adminFormData{Product: product, VariantsText: r.FormValue("variantes"), IsEdit: true, Error: "Erro ao atualizar: " + err.Error()})
		return
	}

	http.Redirect(w, r, "/admin/produtos?flash=Produto+atualizado+com+sucesso", http.StatusFound)
}

func (h *AdminHandler) DeleteProduct(w http.ResponseWriter, r *http.Request) {
	if err := h.repo.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		http.Error(w, "Erro ao remover produto", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/produtos?flash=Produto+removido", http.StatusFound)
}

// UploadImage recebe um arquivo e devolve a URL pública (chamado via fetch no form).
func (h *AdminHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20) // 10 MB
	file, header, err := r.FormFile("imagem")
	if err != nil {
		http.Error(w, "Arquivo inválido", http.StatusBadRequest)
		return
	}
	defer file.Close()

	url, err := h.upload.Upload(file, header)
	if err != nil {
		http.Error(w, "Falha no upload: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(url))
}

// --- helpers ---

func (h *AdminHandler) parseForm(r *http.Request) (*models.Product, string) {
	r.ParseMultipartForm(10 << 20)

	name := strings.TrimSpace(r.FormValue("nome"))
	if name == "" {
		return &models.Product{}, "Nome é obrigatório."
	}
	code := strings.TrimSpace(r.FormValue("codigo"))

	var price float64
	if v := r.FormValue("preco"); v != "" {
		price, _ = parsePrice(v)
	}
	if price <= 0 {
		return &models.Product{}, "Preço inválido."
	}

	images := strings.Fields(r.FormValue("imagens"))

	status := r.FormValue("status")
	if status != "available" && status != "unavailable" {
		status = "available"
	}

	slug := r.FormValue("slug")
	if slug == "" {
		slug = slugify(name)
	}

	var salePrice float64
	if v := r.FormValue("preco_promocional"); v != "" {
		salePrice, _ = parsePrice(v)
	}

	sortOrder := 0
	if v := r.FormValue("ordem"); v != "" {
		fmt.Sscanf(v, "%d", &sortOrder)
	}

	product := &models.Product{
		Code:        code,
		Name:        name,
		Brand:       strings.TrimSpace(r.FormValue("marca")),
		Model:       strings.TrimSpace(r.FormValue("modelo")),
		Description: strings.TrimSpace(r.FormValue("descricao")),
		Price:       price,
		SalePrice:   salePrice,
		Images:      images,
		Status:      status,
		Featured:    r.FormValue("destaque") == "1",
		IsNew:       r.FormValue("is_new") == "1",
		IsSale:      r.FormValue("is_sale") == "1",
		IsLimited:   r.FormValue("is_limited") == "1",
		SortOrder:   sortOrder,
		Slug:        slug,
		Variants:    parseVariants(r.FormValue("variantes")),
	}

	return product, ""
}

// parseVariants interpreta o campo de variantes no formato:
//
//	Branco: 38, 39, 40
//	Preto: 40, 41
func parseVariants(s string) []models.ProductVariant {
	var variants []models.ProductVariant
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		color := strings.TrimSpace(parts[0])
		for _, size := range strings.Split(parts[1], ",") {
			size = strings.TrimSpace(size)
			if size != "" {
				variants = append(variants, models.ProductVariant{
					Color:     color,
					Size:      size,
					Available: true,
				})
			}
		}
	}
	return variants
}

func variantsToText(variants []models.ProductVariant) string {
	colorSizes := map[string][]string{}
	order := []string{}
	seen := map[string]bool{}

	for _, v := range variants {
		if !seen[v.Color] {
			order = append(order, v.Color)
			seen[v.Color] = true
		}
		colorSizes[v.Color] = append(colorSizes[v.Color], v.Size)
	}

	var lines []string
	for _, color := range order {
		lines = append(lines, color+": "+strings.Join(colorSizes[color], ", "))
	}
	return strings.Join(lines, "\n")
}
