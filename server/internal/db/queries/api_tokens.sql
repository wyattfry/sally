-- name: CreateAPIToken :one
insert into api_tokens (user_id, label, token_hash)
values ($1, $2, $3)
returning *;

-- name: ListAPITokens :many
select *
from api_tokens
order by created_at desc;

-- name: GetAPITokenByHash :one
select *
from api_tokens
where token_hash = $1;

-- name: TouchAPITokenLastUsed :exec
update api_tokens set last_used_at = now() where id = $1;

-- name: DeleteAPIToken :exec
delete from api_tokens where id = $1;
