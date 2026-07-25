-- Duplicate active rows exercise the cleanup before the unique partial index is created.
INSERT INTO
  short_codes (id, code, usage, target, created_at, expires_at)
VALUES
  (
    '00000000-0000-0000-0000-000000000011',
    'fixture-code-1',
    'password-reset',
    'roundtrip@example.com',
    '2025-02-08T15:13:00Z',
    '2099-01-01T00:00:00Z'
  ),
  (
    '00000000-0000-0000-0000-000000000012',
    'fixture-code-2',
    'password-reset',
    'roundtrip@example.com',
    '2025-02-08T15:14:00Z',
    '2099-01-01T00:00:00Z'
  ),
  (
    '00000000-0000-0000-0000-000000000013',
    'fixture-code-3',
    'password-reset',
    'roundtrip@example.com',
    '2025-02-08T15:15:00Z',
    '2099-01-01T00:00:00Z'
  );
