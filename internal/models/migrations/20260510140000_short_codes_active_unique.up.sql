/*
The unique index enforces at most one non-deleted short code per (target, usage)
pair, closing a check-then-insert race in the application's ShortCodeInsert path.

The index can only be created once existing rows satisfy it, so this step first
soft-deletes any duplicate active rows the race may have left behind. The most
recent row per (target, usage) is kept; older duplicates are marked deleted with a
distinct comment so they stay auditable. In a healthy deployment this affects zero
rows.
*/
WITH
  ranked AS (
    SELECT
      id,
      ROW_NUMBER() OVER (
        PARTITION BY
          target,
          usage
        ORDER BY
          created_at DESC,
          id DESC
      ) AS rank
    FROM
      short_codes
    WHERE
      deleted_at IS NULL
  )
UPDATE short_codes
SET
  deleted_at = CURRENT_TIMESTAMP,
  deleted_comment = 'duplicate cleanup before unique index'
WHERE
  id IN (
    SELECT
      id
    FROM
      ranked
    WHERE
      rank > 1
  );

/*
Postgres requires partial-index predicates to be IMMUTABLE, so the predicate
can't test `expires_at > now()` directly. The (deleted_at IS NULL) form covers
active conflicts; rows that have naturally expired but aren't yet soft-deleted
also sit in the partial index, but the DAO always runs its discardExpired
soft-delete step before either conflict path, so those stale rows never block a
fresh insert.
*/
CREATE UNIQUE INDEX short_codes_active_target_usage_uniq ON short_codes (target, usage)
WHERE
  deleted_at IS NULL;
