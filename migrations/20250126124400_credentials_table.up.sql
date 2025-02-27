CREATE TABLE credentials
(
    id         uuid PRIMARY KEY            NOT NULL,

    email      TEXT                        NOT NULL UNIQUE CHECK (email <> ''),
    password   TEXT,

    created_at TIMESTAMP(0) WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP(0) WITH TIME ZONE NOT NULL
);

CREATE INDEX credentials_email_idx ON credentials (email);
