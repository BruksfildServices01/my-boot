package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"myboot/internal/models"
)

type TestimonialRepo struct {
	db *pgxpool.Pool
}

func NewTestimonialRepo(db *pgxpool.Pool) *TestimonialRepo {
	return &TestimonialRepo{db: db}
}

func (r *TestimonialRepo) List(ctx context.Context, onlyVisible bool) ([]models.Testimonial, error) {
	q := `SELECT id, type, caption, url, visible, sort_order, created_at FROM testimonials`
	if onlyVisible {
		q += ` WHERE visible = true`
	}
	q += ` ORDER BY sort_order DESC, created_at DESC`

	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.Testimonial
	for rows.Next() {
		var t models.Testimonial
		if err := rows.Scan(&t.ID, &t.Type, &t.Caption, &t.URL, &t.Visible, &t.SortOrder, &t.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, nil
}

func (r *TestimonialRepo) Create(ctx context.Context, t *models.Testimonial) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO testimonials (type, caption, url, visible, sort_order)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at
	`, t.Type, t.Caption, t.URL, t.Visible, t.SortOrder).Scan(&t.ID, &t.CreatedAt)
}

func (r *TestimonialRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM testimonials WHERE id = $1", id)
	return err
}

func (r *TestimonialRepo) ToggleVisible(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "UPDATE testimonials SET visible = NOT visible WHERE id = $1", id)
	return err
}
