UPDATE credentials
SET
  location_consent = ?0
WHERE
  id = ?1
RETURNING
  *;
