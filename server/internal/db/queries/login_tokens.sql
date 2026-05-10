-- name: CreateLoginToken :one
insert into login_tokens (user_id, token_hash)
values ($1, $2)
returning id, user_id, token_hash, expires_at, used_at, created_at;

-- name: GetLoginTokenByHash :one
select id, user_id, token_hash, expires_at, used_at, created_at
from login_tokens
where token_hash = $1
  and used_at is null
  and expires_at > now();

-- name: MarkLoginTokenUsed :exec
update login_tokens set used_at = now() where id = $1;
