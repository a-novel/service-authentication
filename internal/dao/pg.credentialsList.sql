-- id breaks ties on created_at, and created_at does not mutate. Ordering on updated_at let a
-- credential touched between two page queries move across the boundary and be skipped or repeated,
-- and its timestamp(0) precision made same-second rows a tie the planner could order either way.
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
  created_at DESC,
  id DESC
LIMIT
  ?0
OFFSET
  ?1;
