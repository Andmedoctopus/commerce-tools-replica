ALTER TABLE categories
    DROP COLUMN IF EXISTS meta_description,
    DROP COLUMN IF EXISTS meta_title,
    DROP COLUMN IF EXISTS description;
