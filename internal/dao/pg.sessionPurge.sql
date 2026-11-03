-- The sessions inactive the longest go first, through the last_active_at index.
DELETE FROM sessions
WHERE
  id IN (
    SELECT
      id
    FROM
      sessions
    WHERE
      last_active_at < ?0
    ORDER BY
      last_active_at
    LIMIT
      ?1
  );
