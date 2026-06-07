-- +goose Up
CREATE TABLE IF NOT EXISTS posts (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    title TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    description TEXT,
    published_at TIMESTAMPTZ,
    feed_id UUID NOT NULL REFERENCES feed(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_posts_feed_id ON posts (feed_id);
CREATE INDEX IF NOT EXISTS idx_posts_published_at ON posts (published_at DESC);

-- +goose Down
DROP TABLE IF EXISTS posts;
