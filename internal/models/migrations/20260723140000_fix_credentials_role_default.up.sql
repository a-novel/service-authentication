-- The role column defaulted to 'user', which is not a role the application defines — every
-- configured role is auth:*. Rows created under that default resolve to no permissions and rank
-- priority 0, denying those accounts everything with no signal.
-- Point the default at the intended role for a new authenticated account.
ALTER TABLE credentials
ALTER COLUMN role
SET DEFAULT 'auth:user';

-- Remap the rows the old default produced. 'user' meant auth:user; it is the only value the write
-- path could not have produced, since credentialsUpdateRole validates every role it writes against
-- the config.
UPDATE credentials
SET role = 'auth:user'
WHERE
  role = 'user';

-- Refuse any role the application does not know, at write time rather than at read time. The set
-- mirrors internal/config/permissions.config.yaml and must be updated with it; the backfill above
-- runs first so no existing row violates it.
ALTER TABLE credentials
ADD CONSTRAINT credentials_role_check CHECK (
  role IN (
    'auth:anon',
    'auth:user',
    'auth:admin',
    'auth:superadmin'
  )
);
