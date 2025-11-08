CREATE TABLE credentials (
  id uuid PRIMARY KEY NOT NULL,
  email text NOT NULL UNIQUE CHECK (email <> ''),
  password text,
  created_at timestamp(0) with time zone NOT NULL,
  updated_at timestamp(0) with time zone NOT NULL
);
