-- An authorization code is a one-time value a client trades, within seconds, for a token pair in the
-- session of the account platform that issued it. Only its SHA-256 is stored. An exchanged row stays
-- until it expires, so a second exchange finds it and ends the session. The maintenance job then
-- purges it through the expires_at index.
CREATE TABLE authorization_codes (
  /* SHA-256 of the code. The code itself reaches only the client. */
  id bytea PRIMARY KEY NOT NULL CHECK (length(id) = 32),
  /* Client the code was issued to, as the client registry names it. */
  client_id text NOT NULL CHECK (client_id <> ''),
  /* Redirect URI the code was sent to. The exchange names it again. */
  redirect_uri text NOT NULL CHECK (redirect_uri <> ''),
  /* PKCE S256 challenge: the base64url SHA-256 of the client's code verifier. */
  code_challenge text NOT NULL CHECK (code_challenge <> ''),
  /* Account the code signs in. */
  user_id uuid NOT NULL REFERENCES credentials (id) ON DELETE CASCADE,
  /* Session the exchanged token pair joins, the account platform's. */
  session_id text NOT NULL CHECK (session_id <> ''),
  /* When that session ends. The exchanged refresh token stops renewing then. */
  session_expires_at timestamp(0) with time zone NOT NULL,
  /* Last instant the code exchanges. */
  expires_at timestamp(0) with time zone NOT NULL,
  /* When the code was exchanged. Null until then. */
  consumed_at timestamp(0) with time zone
);

CREATE INDEX authorization_codes_expires_at_idx ON authorization_codes (expires_at);

CREATE INDEX authorization_codes_user_id_idx ON authorization_codes (user_id);
