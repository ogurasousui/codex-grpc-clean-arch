INSERT INTO users (email, name, status)
VALUES ('seed-user@example.com', 'Seed User', 'active')
ON CONFLICT (email) DO NOTHING;
