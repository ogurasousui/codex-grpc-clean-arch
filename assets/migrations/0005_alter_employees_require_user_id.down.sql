ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS email TEXT;

ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS last_name TEXT NOT NULL DEFAULT 'unknown';

ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS first_name TEXT NOT NULL DEFAULT 'unknown';

UPDATE employees
   SET last_name = COALESCE(last_name, 'unknown'),
       first_name = COALESCE(first_name, 'unknown');

ALTER TABLE employees
    ALTER COLUMN last_name DROP DEFAULT,
    ALTER COLUMN first_name DROP DEFAULT;

ALTER TABLE employees
    ALTER COLUMN user_id DROP NOT NULL;
