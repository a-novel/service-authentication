-- A refresh token is a stateless JWT, so signing out records its JTI here and a refresh refuses any
-- JTI the table holds. A row is useless once the verifier refuses the token on its own, so a
-- periodic purge deletes rows past expires_at, through the index below.
CREATE TABLE refresh_token_revocations (
  /* JTI of the revoked refresh token. */
  jti text PRIMARY KEY NOT NULL CHECK (jti <> ''),
  /* Last instant the verifier accepts the refresh token: its expiry plus the clock-skew leeway. */
  expires_at timestamp(0) with time zone NOT NULL,
  /* When the refresh token was first revoked. */
  created_at timestamp(0) with time zone NOT NULL
);

CREATE INDEX refresh_token_revocations_expires_at_idx ON refresh_token_revocations (expires_at);
