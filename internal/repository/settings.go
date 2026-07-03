package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsRepo struct {
	db *pgxpool.Pool
}

func NewSettingsRepo(db *pgxpool.Pool) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) Get(ctx context.Context, key string) (string, error) {
	var val string
	err := r.db.QueryRow(ctx, "SELECT value FROM settings WHERE key = $1", key).Scan(&val)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	return val, err
}

func (r *SettingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, key, value)
	return err
}
