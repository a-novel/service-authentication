INSERT INTO
  authorization_codes (
    id,
    user_id,
    client_id,
    redirect_uri,
    redirect_uri_in_request,
    code_challenge,
    code_challenge_method,
    session_id,
    session_expires_at,
    expires_at
  )
VALUES
  (?0, ?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9);
