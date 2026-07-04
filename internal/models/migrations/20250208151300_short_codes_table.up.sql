CREATE TABLE short_codes (
  id uuid PRIMARY KEY NOT NULL,
  /* The encrypted code. The plaintext version is sent to the target. */
  code text NOT NULL,
  /* Action this code authorizes. */
  usage text NOT NULL,
  /* The target this code was issued for. */
  target text NOT NULL,
  /* Opaque payload the code's action needs to complete. */
  data bytea,
  created_at timestamp(0) with time zone NOT NULL,
  expires_at timestamp(0) with time zone NOT NULL,
  /* Soft-deletion timestamp; set to expire a code early, e.g. when compromised. */
  deleted_at timestamp(0) with time zone,
  /* Reason the code was deleted early. */
  deleted_comment text
);

CREATE INDEX short_codes_target_usage_idx ON short_codes (target, usage);

CREATE INDEX short_codes_deleted_idx ON short_codes (deleted_at, expires_at);

CREATE INDEX short_codes_created_at_idx ON short_codes (created_at);
