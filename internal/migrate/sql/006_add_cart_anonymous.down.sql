DROP INDEX IF EXISTS idx_carts_anonymous;

ALTER TABLE carts
    DROP COLUMN IF EXISTS anonymous_id;
