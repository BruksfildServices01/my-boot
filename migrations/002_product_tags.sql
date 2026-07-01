ALTER TABLE products
  ADD COLUMN IF NOT EXISTS is_new      BOOLEAN      NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_sale     BOOLEAN      NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS is_limited  BOOLEAN      NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS sale_price  NUMERIC(10,2),
  ADD COLUMN IF NOT EXISTS sort_order  INTEGER      NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_products_is_new   ON products(is_new);
CREATE INDEX IF NOT EXISTS idx_products_is_sale  ON products(is_sale);
CREATE INDEX IF NOT EXISTS idx_products_sort     ON products(sort_order, created_at DESC);
