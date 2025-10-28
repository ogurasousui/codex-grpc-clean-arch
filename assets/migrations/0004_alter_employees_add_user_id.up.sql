ALTER TABLE employees
    ADD COLUMN IF NOT EXISTS user_id UUID;

ALTER TABLE employees
    ADD CONSTRAINT employees_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_employees_user_id ON employees (user_id);
