INSERT INTO
  credentials (
    id,
    email,
    password,
    created_at,
    updated_at,
    credentials_updated_at,
    role
  )
VALUES
  (?0, ?1, ?2, ?3, ?3, ?3, ?4)
RETURNING
  *;
