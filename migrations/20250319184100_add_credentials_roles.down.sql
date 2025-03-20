DROP INDEX IF EXISTS credentials_role_idx;

ALTER TABLE credentials
DROP COLUMN role;

DROP TYPE IF EXISTS credentials_role;
