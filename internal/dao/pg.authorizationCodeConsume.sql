-- Two exchanges of one code may run at once. The row lock serializes them, and the second then sees
-- consumed_at set and updates nothing.
UPDATE authorization_codes
SET
  consumed_at = ?1
WHERE
  id = ?0
  AND consumed_at IS NULL;
