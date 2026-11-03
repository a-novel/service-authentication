-- A session row lets an account list the devices it signed in on, sign them out, and remove the ones
-- it signed out. Tokens and revocations still carry the session: a session is active while
-- expires_at is ahead and refresh_token_revocations holds no row for its ID. The maintenance job
-- purges the rows inactive for 30 days, through the last_active_at index.
CREATE TABLE sessions (
  /* JTI of the refresh token that opened the session. */
  id uuid PRIMARY KEY NOT NULL,
  /* Account the session signs in. */
  user_id uuid NOT NULL REFERENCES credentials (id) ON DELETE CASCADE,
  /* User-Agent of the device's browser, its first 512 characters. Empty when unknown. */
  user_agent text NOT NULL CHECK (char_length(user_agent) <= 512),
  /* IP address of the device. Null unless the account consents to locations. */
  ip_address inet,
  /* CLDR region code of the device, such as FR. Null unless the account consents to locations. */
  region text,
  /* City of the device. Null unless the account consents to locations. */
  city text,
  created_at timestamp(0) with time zone NOT NULL,
  /* When the session's latest access token was issued. */
  last_active_at timestamp(0) with time zone NOT NULL,
  /* When the session ends: the expiry of the refresh token that opened it. */
  expires_at timestamp(0) with time zone NOT NULL
);

CREATE INDEX sessions_user_id_expires_at_idx ON sessions (user_id, expires_at);

CREATE INDEX sessions_last_active_at_idx ON sessions (last_active_at);

-- An account records the location of its sessions only once it consents. The constant default
-- changes only the catalog.
ALTER TABLE credentials
ADD COLUMN location_consent boolean NOT NULL DEFAULT false;
