-- A repeated revocation keeps the first row, so revoking stays idempotent.
INSERT INTO
  refresh_token_revocations (jti, expires_at, created_at)
VALUES
  (?0, ?1, ?2)
ON CONFLICT (jti) DO NOTHING;
