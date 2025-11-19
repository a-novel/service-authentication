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
ORDER BY
  updated_at DESC
LIMIT
  ?0
OFFSET
  ?1;
