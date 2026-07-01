-- My Boot - Schema inicial

CREATE TABLE IF NOT EXISTS products (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(50)  UNIQUE NOT NULL,
    name        VARCHAR(255) NOT NULL,
    brand       VARCHAR(100) NOT NULL,
    model       VARCHAR(100) NOT NULL,
    description TEXT         NOT NULL DEFAULT '',
    price       NUMERIC(10, 2) NOT NULL,
    images      TEXT[]       NOT NULL DEFAULT '{}',
    status      VARCHAR(20)  NOT NULL DEFAULT 'available'
                    CHECK (status IN ('available', 'unavailable')),
    featured    BOOLEAN      NOT NULL DEFAULT FALSE,
    slug        VARCHAR(255) UNIQUE NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS product_variants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    color       VARCHAR(100) NOT NULL,
    size        VARCHAR(20)  NOT NULL,
    available   BOOLEAN      NOT NULL DEFAULT TRUE,
    UNIQUE (product_id, color, size)
);

CREATE INDEX IF NOT EXISTS idx_products_status   ON products(status);
CREATE INDEX IF NOT EXISTS idx_products_brand    ON products(brand);
CREATE INDEX IF NOT EXISTS idx_products_featured ON products(featured DESC, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_products_slug     ON products(slug);
CREATE INDEX IF NOT EXISTS idx_variants_product  ON product_variants(product_id);
CREATE INDEX IF NOT EXISTS idx_variants_color    ON product_variants(color);
CREATE INDEX IF NOT EXISTS idx_variants_size     ON product_variants(size);
