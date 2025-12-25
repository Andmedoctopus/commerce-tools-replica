ALTER TABLE carts
    ADD COLUMN anonymous_id UUID;

CREATE INDEX IF NOT EXISTS idx_carts_anonymous ON carts(anonymous_id);
