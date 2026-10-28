-- A short code's mail waits here until a send delivers it, so a slow relay, a network warm-up or a
-- restart no longer loses it. The plaintext code is never stored: an attempt without it renews the
-- code in place. An entry ends with a delivery, a refusal, or the expiry of its first code.
CREATE TABLE mail_outbox (
  /* Short code the mail carries. */
  short_code_id uuid PRIMARY KEY NOT NULL REFERENCES short_codes (id),
  /* Usage of the short code, which selects the mail. */
  usage text NOT NULL CHECK (usage <> ''),
  /* Target of the short code, which the mail's link carries. */
  target text NOT NULL CHECK (target <> ''),
  /* Address the mail goes to. */
  recipient text NOT NULL CHECK (recipient <> ''),
  /* Language of the mail. */
  lang text NOT NULL CHECK (lang <> ''),
  /* Attempts claimed so far. Their count sets the delay before the next one. */
  attempts integer NOT NULL DEFAULT 0,
  /* When the next attempt may start. A claim moves it past the attempt, as a lease. */
  due_at timestamp(0) with time zone NOT NULL,
  /* Expiry of the first code. Retries stop there. */
  expires_at timestamp(0) with time zone NOT NULL
);

CREATE INDEX mail_outbox_due_at_idx ON mail_outbox (due_at);
