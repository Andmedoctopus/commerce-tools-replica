ALTER TABLE customers
    DROP COLUMN IF EXISTS billing_address_ids,
    DROP COLUMN IF EXISTS shipping_address_ids,
    DROP COLUMN IF EXISTS default_billing_address_id,
    DROP COLUMN IF EXISTS default_shipping_address_id,
    DROP COLUMN IF EXISTS addresses,
    DROP COLUMN IF EXISTS date_of_birth,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS first_name;
