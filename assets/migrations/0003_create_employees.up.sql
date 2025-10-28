CREATE TABLE IF NOT EXISTS employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_code TEXT NOT NULL,
    email TEXT,
    last_name TEXT NOT NULL,
    first_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    hired_at DATE,
    terminated_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT employees_company_code_unique UNIQUE (company_id, employee_code),
    CONSTRAINT employees_terminated_after_hired CHECK (
        terminated_at IS NULL OR hired_at IS NULL OR terminated_at >= hired_at
    )
);

CREATE INDEX IF NOT EXISTS idx_employees_company_id_status ON employees (company_id, status);
