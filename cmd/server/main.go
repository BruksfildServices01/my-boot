package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"myboot/internal/handlers"
	"myboot/internal/repository"
	"myboot/internal/session"
)

func main() {
	_ = godotenv.Load()

	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("banco de dados:", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("ping banco de dados:", err)
	}
	log.Println("Conectado ao banco de dados.")

	sessions := session.NewStore(os.Getenv("COOKIE_SECURE") == "true")
	repo := repository.NewProductRepo(pool)
	testimonialRepo := repository.NewTestimonialRepo(pool)
	settingsRepo := repository.NewSettingsRepo(pool)
	upload := handlers.NewUploadHandler()

	catalog := handlers.NewCatalogHandler(repo, testimonialRepo, settingsRepo)
	admin := handlers.NewAdminHandler(repo, testimonialRepo, settingsRepo, sessions, upload)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CleanPath)
	r.Use(middleware.Compress(5))

	// arquivos estáticos com cache de 1 ano (URLs têm ?v= para invalidar)
	staticFS := http.FileServer(http.Dir("web/static"))
	r.Handle("/static/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.StripPrefix("/static/", staticFS).ServeHTTP(w, r)
	}))

	// vitrine pública
	r.Get("/", catalog.Catalog)
	r.Get("/produto/{slug}", catalog.Product)
	r.Get("/my-boot", catalog.MyBoot)

	// admin — login público
	r.Get("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/produtos", http.StatusFound)
	})
	r.Get("/admin/login", admin.Login)
	r.Post("/admin/login", admin.LoginPost)
	r.Post("/admin/logout", admin.Logout)

	// admin — rotas protegidas
	r.Group(func(r chi.Router) {
		r.Use(admin.AuthMiddleware)
		r.Get("/admin/produtos", admin.Products)
		r.Get("/admin/produtos/novo", admin.NewProduct)
		r.Post("/admin/produtos/criar", admin.CreateProduct)
		r.Get("/admin/produtos/{id}/editar", admin.EditProduct)
		r.Post("/admin/produtos/{id}/atualizar", admin.UpdateProduct)
		r.Post("/admin/produtos/{id}/deletar", admin.DeleteProduct)
		r.Post("/admin/upload", admin.UploadImage)
		r.Get("/admin/myboot", admin.Testimonials)
		r.Get("/admin/myboot/novo", admin.NewTestimonial)
		r.Post("/admin/myboot/criar", admin.CreateTestimonial)
		r.Post("/admin/myboot/{id}/deletar", admin.DeleteTestimonial)
		r.Post("/admin/myboot/{id}/toggle", admin.ToggleTestimonial)
		r.Post("/admin/myboot-toggle", admin.ToggleMyBoot)
	})

	addr := ":" + getenv("PORT", "8080")
	log.Printf("Servidor rodando em http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
