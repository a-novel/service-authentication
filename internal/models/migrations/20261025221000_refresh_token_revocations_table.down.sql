-- Dropping the table forgets every revocation: refresh tokens revoked before the rollback renew
-- their sessions again until they expire.
DROP INDEX IF EXISTS refresh_token_revocations_expires_at_idx;

DROP TABLE IF EXISTS refresh_token_revocations;
