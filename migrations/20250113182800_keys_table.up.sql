CREATE TABLE keys
(
  id uuid PRIMARY KEY NOT NULL,

  /*
      Contains the ciphered JSON Web Key representation of the private key,
      as a string.
    */
  private_key text NOT NULL CHECK (private_key <> ''),
  /* Contains the raw JSON Web key public representation, if available. */
  public_key text,
  /* Group keys of similar usage */
  usage text NOT NULL,

  created_at timestamp (0) with time zone NOT NULL,
  /* Sets an expiration date for the key. */
  expires_at timestamp (0) with time zone NOT NULL,
  /* Use this field to expire a key early, in case it was compromised. */
  deleted_at timestamp (0) with time zone,
  /* Extra information about the deprecation of the key. */
  deleted_comment text
);

CREATE INDEX keys_usage_idx ON keys (usage);

CREATE VIEW active_keys AS
(
  SELECT *
  FROM keys
  WHERE COALESCE(deleted_at, expires_at) > CURRENT_TIMESTAMP(0)
  ORDER BY id DESC
);
