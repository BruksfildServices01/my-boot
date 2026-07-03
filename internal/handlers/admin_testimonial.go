package handlers

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"myboot/internal/models"
)

type testimonialsListData struct {
	Items   []models.Testimonial
	Flash   string
	Enabled bool
}

func (h *AdminHandler) Testimonials(w http.ResponseWriter, r *http.Request) {
	items, err := h.testimonialRepo.List(r.Context(), false)
	if err != nil {
		http.Error(w, "Erro ao carregar feedbacks", http.StatusInternalServerError)
		return
	}
	val, _ := h.settingsRepo.Get(r.Context(), "myboot_enabled")
	render(w, h.testimonialsTmpl, testimonialsListData{
		Items:   items,
		Flash:   r.URL.Query().Get("flash"),
		Enabled: val != "false",
	})
}

func (h *AdminHandler) ToggleMyBoot(w http.ResponseWriter, r *http.Request) {
	val, _ := h.settingsRepo.Get(r.Context(), "myboot_enabled")
	next := "false"
	if val == "false" {
		next = "true"
	}
	h.settingsRepo.Set(r.Context(), "myboot_enabled", next)
	http.Redirect(w, r, "/admin/myboot", http.StatusFound)
}

func (h *AdminHandler) NewTestimonial(w http.ResponseWriter, r *http.Request) {
	render(w, h.testimonialFormTmpl, nil)
}

func (h *AdminHandler) CreateTestimonial(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	mediaType := r.FormValue("tipo")
	if mediaType != "video" && mediaType != "image" {
		render(w, h.testimonialFormTmpl, map[string]string{"Error": "Tipo inválido."})
		return
	}

	var url string

	if mediaType == "video" {
		url = strings.TrimSpace(r.FormValue("youtube_url"))
		if url == "" {
			render(w, h.testimonialFormTmpl, map[string]string{"Error": "URL do YouTube é obrigatória."})
			return
		}
	} else {
		file, header, err := r.FormFile("imagem")
		if err != nil {
			render(w, h.testimonialFormTmpl, map[string]string{"Error": "Selecione uma imagem."})
			return
		}
		defer file.Close()
		url, err = h.upload.Upload(file, header)
		if err != nil {
			render(w, h.testimonialFormTmpl, map[string]string{"Error": "Falha no upload: " + err.Error()})
			return
		}
	}

	t := &models.Testimonial{
		Type:    models.MediaType(mediaType),
		Caption: strings.TrimSpace(r.FormValue("legenda")),
		URL:     url,
		Visible: r.FormValue("visivel") == "1",
	}

	if err := h.testimonialRepo.Create(r.Context(), t); err != nil {
		render(w, h.testimonialFormTmpl, map[string]string{"Error": "Erro ao salvar: " + err.Error()})
		return
	}

	http.Redirect(w, r, "/admin/myboot?flash=Item+adicionado+com+sucesso", http.StatusFound)
}

func (h *AdminHandler) DeleteTestimonial(w http.ResponseWriter, r *http.Request) {
	if err := h.testimonialRepo.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		http.Error(w, "Erro ao remover", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/myboot?flash=Item+removido", http.StatusFound)
}

func (h *AdminHandler) ToggleTestimonial(w http.ResponseWriter, r *http.Request) {
	if err := h.testimonialRepo.ToggleVisible(r.Context(), chi.URLParam(r, "id")); err != nil {
		http.Error(w, "Erro ao atualizar", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/myboot", http.StatusFound)
}
