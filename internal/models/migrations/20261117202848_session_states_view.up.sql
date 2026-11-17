-- One definition of a logged-out session for every query that asks whether a session is logged
-- out: it ended, or a revocation holds its ID, which a sign-out writes. The view selects sessions.* as the table
-- stands, so a migration that adds a session column recreates it.
CREATE VIEW session_states AS
SELECT
  sessions.*,
  (
    sessions.expires_at <= CURRENT_TIMESTAMP
    OR EXISTS (
      SELECT
        1
      FROM
        refresh_token_revocations
      WHERE
        refresh_token_revocations.jti = sessions.id::text
    )
  ) AS logged_out
FROM
  sessions;
