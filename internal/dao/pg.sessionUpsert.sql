-- A new ID opens the session. A known one keeps its account, its start and its end, so the session
-- of an older release gets its row on its next renewal.
INSERT INTO
  sessions (
    id,
    user_id,
    user_agent,
    ip_address,
    region,
    city,
    created_at,
    last_active_at,
    expires_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?6, ?7)
ON CONFLICT (id) DO UPDATE
SET
  user_agent = EXCLUDED.user_agent,
  ip_address = EXCLUDED.ip_address,
  region = EXCLUDED.region,
  city = EXCLUDED.city,
  last_active_at = EXCLUDED.last_active_at;
