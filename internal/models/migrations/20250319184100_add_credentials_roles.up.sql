ALTER TABLE credentials
ADD COLUMN role text NOT NULL DEFAULT 'user';

CREATE INDEX credentials_role_idx ON credentials (role);
