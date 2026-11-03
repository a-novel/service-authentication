UPDATE sessions
SET
  ip_address = NULL,
  region = NULL,
  city = NULL
WHERE
  user_id = ?0;
