-- A fixture runs after its matching migration and seeds data for the next migration.
INSERT INTO
  credentials (id, email, password, created_at, updated_at)
VALUES
  (
    '00000000-0000-0000-0000-000000000001',
    'roundtrip@example.com',
    NULL,
    '2025-01-26T12:44:00Z',
    '2025-01-26T12:44:00Z'
  );
