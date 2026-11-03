-- A session of an older release gets its row, with no device, on its next access token.
INSERT INTO
  sessions (
    id,
    user_id,
    user_agent,
    created_at,
    last_active_at,
    expires_at
  )
VALUES
  (?0, ?1, '', ?2, ?2, ?3)
ON CONFLICT (id) DO UPDATE
SET
  last_active_at = EXCLUDED.last_active_at;
