INSERT INTO users (email, name, status, created_at, updated_at)
SELECT e.email,
       CONCAT(e.last_name, ' ', e.first_name) AS name,
       e.status,
       COALESCE(e.created_at, NOW()),
       COALESCE(e.updated_at, NOW())
  FROM employees e
  LEFT JOIN users u ON u.email = e.email
 WHERE e.email IS NOT NULL
   AND u.id IS NULL
ON CONFLICT (email) DO NOTHING;

INSERT INTO users (email, name, status, created_at, updated_at)
SELECT CONCAT('employee-', e.id, '@placeholder.local') AS email,
       CONCAT(e.last_name, ' ', e.first_name) AS name,
       e.status,
       COALESCE(e.created_at, NOW()),
       COALESCE(e.updated_at, NOW())
  FROM employees e
 WHERE e.email IS NULL
ON CONFLICT (email) DO NOTHING;

UPDATE employees e
   SET user_id = u.id
  FROM users u
 WHERE e.user_id IS NULL
   AND e.email IS NOT NULL
   AND LOWER(u.email) = LOWER(e.email);

UPDATE employees e
   SET user_id = u.id
  FROM users u
 WHERE e.user_id IS NULL
   AND e.email IS NULL
   AND u.email = CONCAT('employee-', e.id, '@placeholder.local');

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM employees WHERE user_id IS NULL) THEN
        RAISE EXCEPTION 'employees.user_id must be populated before dropping personal columns';
    END IF;
END $$;

ALTER TABLE employees
    ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE employees
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS last_name,
    DROP COLUMN IF EXISTS first_name;
