-- Canonical database schema managed via Atlas in declarative SQL form.
-- Update this file to represent the desired state of the database schema.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);

CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    code TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_companies_status ON companies (status);

CREATE TABLE employees (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID NOT NULL REFERENCES companies(id) ON DELETE CASCADE,
    employee_code TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    hired_at DATE,
    terminated_at DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT employees_company_code_unique UNIQUE (company_id, employee_code),
    CONSTRAINT employees_terminated_after_hired CHECK (
        terminated_at IS NULL OR hired_at IS NULL OR terminated_at >= hired_at
    )
);

CREATE INDEX idx_employees_company_id_status ON employees (company_id, status);
CREATE INDEX idx_employees_user_id ON employees (user_id);

-- Migration metadata managed by golang-migrate.
CREATE TABLE schema_migrations (
    version BIGINT NOT NULL PRIMARY KEY,
    dirty BOOLEAN NOT NULL
);
