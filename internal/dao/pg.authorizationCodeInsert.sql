INSERT INTO
  authorization_codes (
    id,
    user_id,
    client_id,
    redirect_uri,
    code_challenge,
    session_id,
    session_expires_at,
    expires_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?7);
