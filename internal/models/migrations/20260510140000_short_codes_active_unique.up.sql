/*
Close the (target, usage) race in ShortCodeInsert by enforcing at the
database level what the application has always intended: at most one
non-deleted short code per (target, usage) pair.

Pre-cleanup soft-deletes any duplicate active rows the racy check-then-insert
may have introduced before the constraint can be applied. The most recent
row per (target, usage) is kept; older duplicates are marked deleted with a
distinct comment so they're auditable. In a healthy deployment this affects
zero rows.
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
can't include `expires_at > now()` directly. The (deleted_at IS NULL) form
covers active conflicts; naturally-expired-but-not-yet-deleted rows are also
in the partial index, but the dao retires them before every insert — the
Override=true path's discardConflicts soft-deletes every not-yet-deleted row
for the pair, and the Override=false path runs a dedicated discardExpired
step — so those stale rows never block a fresh insert.
*/
CREATE UNIQUE INDEX short_codes_active_target_usage_uniq ON short_codes (target, usage)
WHERE
  deleted_at IS NULL;
