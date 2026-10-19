-- Pages sort on created_at, which never changes, so a row cannot cross a page boundary between two
-- queries. id breaks the ties of rows created in the same second.
SELECT
  id,
  email,
  role,
  created_at,
  updated_at
FROM
  credentials
WHERE
  (
    (?2) IS NULL -- If no role is provided (empty array), don't filter on roles.
    OR role IN (?2)
  )
  AND (
    (?3) IS NULL
    OR email IN (?3)
  )
ORDER BY
  created_at DESC,
  id DESC
LIMIT
  ?0
OFFSET
  ?1;
