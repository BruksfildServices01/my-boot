package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"myboot/internal/models"
)

type ProductRepo struct {
	db *pgxpool.Pool
}

func NewProductRepo(db *pgxpool.Pool) *ProductRepo {
	return &ProductRepo{db: db}
}

// GenerateCode cria um código único baseado na marca: NIK001, NIK002, ADI001…
func (r *ProductRepo) GenerateCode(ctx context.Context, brand string) (string, error) {
	prefix := brandPrefix(brand)

	var max int
	r.db.QueryRow(ctx, `
		SELECT COALESCE(MAX(
			CAST(NULLIF(REGEXP_REPLACE(code, '[^0-9]', '', 'g'), '') AS INTEGER)
		), 0)
		FROM products
		WHERE code LIKE $1
	`, prefix+"%").Scan(&max)

	code := fmt.Sprintf("%s%03d", prefix, max+1)

	var exists bool
	r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM products WHERE code = $1)", code).Scan(&exists)
	if exists {
		code = fmt.Sprintf("%s%03d", prefix, max+2)
	}
	return code, nil
}

func brandPrefix(brand string) string {
	brand = strings.ToUpper(strings.TrimSpace(brand))
	replacer := strings.NewReplacer(
		"Ã", "A", "Á", "A", "À", "A", "Â", "A",
		"É", "E", "Ê", "E", "Í", "I", "Õ", "O",
		"Ó", "O", "Ô", "O", "Ú", "U", "Ç", "C",
	)
	brand = replacer.Replace(brand)
	var b strings.Builder
	for _, r := range brand {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	for len(s) < 3 {
		s += "X"
	}
	return s[:3]
}

const productCols = `
	p.id, p.code, p.name, p.brand, p.model, p.description,
	p.price, p.sale_price, p.images, p.status, p.featured,
	p.is_new, p.is_sale, p.is_limited, p.sort_order, p.slug,
	p.created_at, p.updated_at`

func scanProduct(row pgx.Row) (*models.Product, error) {
	var p models.Product
	var salePrice *float64
	err := row.Scan(
		&p.ID, &p.Code, &p.Name, &p.Brand, &p.Model, &p.Description,
		&p.Price, &salePrice, &p.Images, &p.Status, &p.Featured,
		&p.IsNew, &p.IsSale, &p.IsLimited, &p.SortOrder, &p.Slug,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if salePrice != nil {
		p.SalePrice = *salePrice
	}
	return &p, nil
}

func (r *ProductRepo) List(ctx context.Context, f models.ProductFilter) ([]models.Product, error) {
	args := []any{}
	conds := []string{}
	i := 1

	if f.Search != "" {
		conds = append(conds, fmt.Sprintf(
			"(p.name ILIKE $%d OR p.brand ILIKE $%d OR p.model ILIKE $%d)", i, i, i,
		))
		args = append(args, "%"+f.Search+"%")
		i++
	}
	if f.Brand != "" {
		conds = append(conds, fmt.Sprintf("p.brand = $%d", i))
		args = append(args, f.Brand)
		i++
	}
	if f.OnlyAvailable {
		conds = append(conds, "p.status = 'available'")
	}
	if f.OnlyNew {
		conds = append(conds, "p.is_new = true")
	}
	if f.OnlySale {
		conds = append(conds, "p.is_sale = true")
	}
	if f.OnlyLimited {
		conds = append(conds, "p.is_limited = true")
	}
	if f.OnlyFeatured {
		conds = append(conds, "p.featured = true")
	}
	if f.MinPrice > 0 {
		conds = append(conds, fmt.Sprintf("p.price >= $%d", i))
		args = append(args, f.MinPrice)
		i++
	}
	if f.MaxPrice > 0 {
		conds = append(conds, fmt.Sprintf("p.price <= $%d", i))
		args = append(args, f.MaxPrice)
		i++
	}
	if f.Color != "" {
		conds = append(conds, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id AND v.color ILIKE $%d AND v.available = true)", i,
		))
		args = append(args, "%"+f.Color+"%")
		i++
	}
	if f.Size != "" {
		conds = append(conds, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM product_variants v WHERE v.product_id = p.id AND v.size = $%d AND v.available = true)", i,
		))
		args = append(args, f.Size)
		i++
	}

	where := "1=1"
	if len(conds) > 0 {
		where = strings.Join(conds, " AND ")
	}

	query := fmt.Sprintf(`
		SELECT %s FROM products p
		WHERE %s
		ORDER BY
			CASE WHEN p.featured     THEN 0 ELSE 1 END,
			CASE WHEN p.sort_order > 0 THEN p.sort_order ELSE 99999 END,
			p.is_new  DESC,
			p.is_sale DESC,
			p.created_at DESC
	`, productCols, where)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		var salePrice *float64
		if err := rows.Scan(
			&p.ID, &p.Code, &p.Name, &p.Brand, &p.Model, &p.Description,
			&p.Price, &salePrice, &p.Images, &p.Status, &p.Featured,
			&p.IsNew, &p.IsSale, &p.IsLimited, &p.SortOrder, &p.Slug,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if salePrice != nil {
			p.SalePrice = *salePrice
		}
		products = append(products, p)
	}
	return products, nil
}

func (r *ProductRepo) GetBySlug(ctx context.Context, slug string) (*models.Product, error) {
	p, err := scanProduct(r.db.QueryRow(ctx,
		"SELECT "+productCols+" FROM products p WHERE p.slug = $1", slug,
	))
	if err != nil || p == nil {
		return nil, err
	}
	p.Variants, err = r.loadVariants(ctx, p.ID)
	return p, err
}

func (r *ProductRepo) GetByID(ctx context.Context, id string) (*models.Product, error) {
	p, err := scanProduct(r.db.QueryRow(ctx,
		"SELECT "+productCols+" FROM products p WHERE p.id = $1", id,
	))
	if err != nil || p == nil {
		return nil, err
	}
	p.Variants, err = r.loadVariants(ctx, p.ID)
	return p, err
}

func (r *ProductRepo) Create(ctx context.Context, p *models.Product) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var salePrice *float64
	if p.SalePrice > 0 {
		salePrice = &p.SalePrice
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO products
			(code, name, brand, model, description, price, sale_price, images,
			 status, featured, is_new, is_sale, is_limited, sort_order, slug)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, created_at, updated_at
	`, p.Code, p.Name, p.Brand, p.Model, p.Description, p.Price, salePrice, p.Images,
		p.Status, p.Featured, p.IsNew, p.IsSale, p.IsLimited, p.SortOrder, p.Slug,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return err
	}
	if err := insertVariants(ctx, tx, p.ID, p.Variants); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ProductRepo) Update(ctx context.Context, p *models.Product) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var salePrice *float64
	if p.SalePrice > 0 {
		salePrice = &p.SalePrice
	}

	_, err = tx.Exec(ctx, `
		UPDATE products SET
			code=$1, name=$2, brand=$3, model=$4, description=$5,
			price=$6, sale_price=$7, images=$8, status=$9, featured=$10,
			is_new=$11, is_sale=$12, is_limited=$13, sort_order=$14, slug=$15,
			updated_at=NOW()
		WHERE id=$16
	`, p.Code, p.Name, p.Brand, p.Model, p.Description,
		p.Price, salePrice, p.Images, p.Status, p.Featured,
		p.IsNew, p.IsSale, p.IsLimited, p.SortOrder, p.Slug, p.ID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM product_variants WHERE product_id = $1", p.ID); err != nil {
		return err
	}
	if err := insertVariants(ctx, tx, p.ID, p.Variants); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *ProductRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM products WHERE id = $1", id)
	return err
}

func (r *ProductRepo) ListBrands(ctx context.Context) ([]string, error) {
	return r.listDistinct(ctx, "SELECT DISTINCT brand FROM products ORDER BY brand")
}

func (r *ProductRepo) ListColors(ctx context.Context) ([]string, error) {
	return r.listDistinct(ctx, "SELECT DISTINCT color FROM product_variants ORDER BY color")
}

func (r *ProductRepo) ListSizes(ctx context.Context) ([]string, error) {
	return r.listDistinct(ctx, `
		SELECT DISTINCT size FROM product_variants
		ORDER BY
			CASE WHEN size ~ '^[0-9]+(\.[0-9]+)?$'
				THEN CAST(size AS NUMERIC) ELSE 9999 END,
			size
	`)
}

func (r *ProductRepo) PriceRange(ctx context.Context) (min, max float64) {
	r.db.QueryRow(ctx, "SELECT COALESCE(MIN(price),0), COALESCE(MAX(price),0) FROM products").Scan(&min, &max)
	return
}

// --- helpers ---

func (r *ProductRepo) loadVariants(ctx context.Context, productID string) ([]models.ProductVariant, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, product_id, color, size, available
		FROM product_variants WHERE product_id = $1
		ORDER BY color, size
	`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []models.ProductVariant
	for rows.Next() {
		var v models.ProductVariant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.Color, &v.Size, &v.Available); err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}
	return variants, nil
}

func (r *ProductRepo) listDistinct(ctx context.Context, q string) ([]string, error) {
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, nil
}

func insertVariants(ctx context.Context, tx pgx.Tx, productID string, variants []models.ProductVariant) error {
	for _, v := range variants {
		if _, err := tx.Exec(ctx, `
			INSERT INTO product_variants (product_id, color, size, available)
			VALUES ($1, $2, $3, $4)
		`, productID, v.Color, v.Size, v.Available); err != nil {
			return err
		}
	}
	return nil
}
