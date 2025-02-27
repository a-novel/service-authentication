CREATE TABLE short_codes
(
  id uuid PRIMARY KEY NOT NULL,

  /* The encrypted code. Clear version of this code is sent to the target. */
  code text NOT NULL,

  /* Action this short code is intended for. */
  usage text NOT NULL,
  /* Information about the target this code was issued for. */
  target text NOT NULL,

  /* Data that is required by the action requiring the short code. */
  data bytea,

  created_at timestamp (0) with time zone NOT NULL,
  /* Sets an expiration date for the short code. */
  expires_at timestamp (0) with time zone NOT NULL,
  /*
    Use this field to expire a short code early, in case it
    was compromised.
  */
  deleted_at timestamp (0) with time zone,
  /* Extra information about the deprecation of the short code. */
  deleted_comment text
);

CREATE INDEX short_codes_target_usage_idx ON short_codes (target, usage);

CREATE VIEW active_short_codes AS
(
  SELECT *
  FROM short_codes
  WHERE COALESCE(deleted_at, expires_at) >= CLOCK_TIMESTAMP()
);

/*
  Prevent insertion if some unexpired short code with the same usage
  and target already exists.
  We cannot use an index because unique constraint depends on non-immutable
  time constraint.
*/
CREATE FUNCTION CHECK_UNIQUE_ACTIVE_SHORT_CODES()
RETURNS trigger AS
$$
BEGIN
  IF EXISTS (SELECT 1 FROM active_short_codes WHERE target = NEW.target AND usage = NEW.usage) THEN
    RAISE unique_violation USING MESSAGE = 'Short code already exists for this target and usage.';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER unique_active_short_codes
BEFORE INSERT
ON short_codes
FOR EACH ROW
EXECUTE FUNCTION CHECK_UNIQUE_ACTIVE_SHORT_CODES();
