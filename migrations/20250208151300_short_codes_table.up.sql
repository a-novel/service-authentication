CREATE TABLE short_codes
(
    id              uuid PRIMARY KEY            NOT NULL,

    /* The encrypted code. Clear version of this code is sent to the target. */
    code            TEXT                        NOT NULL,

    /* Action this short code is intended for. */
    usage           TEXT                        NOT NULL,
    /* Information about the target this code was issued for. */
    target          TEXT                        NOT NULL,

    /* Data that is required by the action requiring the short code. */
    data            BYTEA,

    created_at      TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    /* Sets an expiration date for the short code. */
    expires_at      TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    /* Use this field to expire a short code early, in case it was compromised. */
    deleted_at      TIMESTAMP(0) WITH TIME ZONE,
    /* Extra information about the deprecation of the short code. */
    deleted_comment TEXT
);

CREATE VIEW active_short_codes AS
(
SELECT *
FROM short_codes
WHERE COALESCE(deleted_at, expires_at) >= clock_timestamp()
    );

/*
    Prevent insertion if some unexpired short code with the same usage and target already exists.
    We cannot use an index because unique constraint depends on non-immutable time constraint.
*/
CREATE FUNCTION check_unique_active_short_codes()
    RETURNS TRIGGER AS
$$
BEGIN
    IF EXISTS (SELECT 1
               FROM active_short_codes
               WHERE target = NEW.target
                 AND usage = NEW.usage) THEN
        RAISE unique_violation USING MESSAGE = 'Short code already exists for this target and usage.';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER unique_active_short_codes
    BEFORE INSERT
    ON short_codes
    FOR EACH ROW
EXECUTE FUNCTION check_unique_active_short_codes();
