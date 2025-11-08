INSERT INTO
  short_codes (
    id,
    code,
    usage,
    target,
    data,
    created_at,
    expires_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6)
RETURNING
  *;
