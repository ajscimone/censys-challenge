CREATE TABLE share_links(
    id SERIAL PRIMARY KEY,
    token TEXT NOT NULL UNIQUE,
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    access_count INTEGER NOT NULL DEFAULT 0,
    created_by INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_share_links_token ON share_links(token);
CREATE INDEX idx_share_links_collection_id ON share_links(collection_id);
