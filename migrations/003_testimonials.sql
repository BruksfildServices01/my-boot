CREATE TABLE IF NOT EXISTS testimonials (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    type       TEXT        NOT NULL CHECK (type IN ('video', 'image')),
    caption    TEXT        NOT NULL DEFAULT '',
    url        TEXT        NOT NULL,
    visible    BOOLEAN     NOT NULL DEFAULT true,
    sort_order INT         NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
