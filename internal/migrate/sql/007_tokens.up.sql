CREATE TABLE IF NOT EXISTS tokens (
    token TEXT PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    anonymous_id TEXT,
    kind TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (
        (customer_id IS NOT NULL AND anonymous_id IS NULL)
        OR (customer_id IS NULL AND anonymous_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_tokens_project ON tokens(project_id);
CREATE INDEX IF NOT EXISTS idx_tokens_customer ON tokens(customer_id);
CREATE INDEX IF NOT EXISTS idx_tokens_anonymous ON tokens(anonymous_id);
CREATE INDEX IF NOT EXISTS idx_tokens_expires_at ON tokens(expires_at);
