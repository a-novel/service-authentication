-- A repeated revocation keeps the first row, so revoking stays idempotent.
INSERT INTO
  refresh_token_revocations (jti, expires_at, created_at)
SELECT
  data.jti,
  ?1,
  ?2
FROM
  unnest(ARRAY[?0]::text[]) AS data (jti)
ON CONFLICT (jti) DO NOTHING;
