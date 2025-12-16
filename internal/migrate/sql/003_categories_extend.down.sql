ALTER TABLE categories
    DROP COLUMN IF EXISTS parent_key,
    DROP COLUMN IF EXISTS order_hint,
    DROP COLUMN IF EXISTS slug;
