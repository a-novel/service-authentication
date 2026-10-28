INSERT INTO
  mail_outbox (
    short_code_id,
    usage,
    target,
    recipient,
    lang,
    due_at,
    expires_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6);
