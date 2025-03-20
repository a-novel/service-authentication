CREATE TYPE credentials_role AS ENUM ('super_admin', 'admin', 'user');

ALTER TABLE credentials
ADD COLUMN role credentials_role NOT NULL DEFAULT 'user';

CREATE INDEX credentials_role_idx ON credentials (role);
