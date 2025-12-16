ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS description TEXT,
    ADD COLUMN IF NOT EXISTS meta_title TEXT,
    ADD COLUMN IF NOT EXISTS meta_description TEXT;
