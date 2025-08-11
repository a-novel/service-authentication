CREATE TABLE short_codes (
  id uuid PRIMARY KEY NOT NULL,
  /* The encrypted code. Clear version of this code is sent to the target. */
  code text NOT NULL,
  /* Action this short code is intended for. */
  usage text NOT NULL,
  /* Information about the target this code was issued for. */
  target text NOT NULL,
  /* Data that is required by the action requiring the short code. */
  data bytea,
  created_at timestamp(0) with time zone NOT NULL,
  /* Sets an expiration date for the short code. */
  expires_at timestamp(0) with time zone NOT NULL,
  /*
  Use this field to expire a short code early, in case it
  was compromised.
  */
  deleted_at timestamp(0) with time zone,
  /* Extra information about the deprecation of the short code. */
  deleted_comment text
);

CREATE INDEX short_codes_target_usage_idx ON short_codes (target, usage);

CREATE INDEX short_codes_deleted_idx ON short_codes (deleted_at, expires_at);

CREATE INDEX short_codes_created_at_idx ON short_codes (created_at);
