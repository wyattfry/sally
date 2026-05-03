-- name: CreateUser :one
insert into users (email, name)
values ($1, $2)
on conflict (email) do update
set name = excluded.name,
    updated_at = now()
returning *;

-- name: GetUser :one
select *
from users
where id = $1;
