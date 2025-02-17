CREATE TABLE keys
(
    id              uuid PRIMARY KEY            NOT NULL,

    /* Contains the ciphered JSON Web Key representation of the private key, as a string. */
    private_key     TEXT                        NOT NULL CHECK (private_key <> ''),
    /* Contains the raw JSON Web key public representation, if available. */
    public_key      TEXT,
    /* Group keys of similar usage */
    usage           TEXT                        NOT NULL,

    created_at      TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    /* Sets an expiration date for the key. */
    expires_at      TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    /* Use this field to expire a key early, in case it was compromised. */
    deleted_at      TIMESTAMP(0) WITH TIME ZONE,
    /* Extra information about the deprecation of the key. */
    deleted_comment TEXT
);

CREATE VIEW active_keys AS
(
SELECT *
FROM keys
WHERE COALESCE(deleted_at, expires_at) > current_timestamp(0)
    );
