-- name: CreateProject :one
insert into projects (owner_user_id, name, address)
values ($1, $2, $3)
returning *;

-- name: GetProject :one
select *
from projects
where id = $1;

-- name: ListProjectsByOwner :many
select *
from projects
where owner_user_id = $1
order by updated_at desc, created_at desc;

-- name: UpdateProject :one
update projects
set name = $2,
    address = $3,
    updated_at = now()
where id = $1
returning *;

-- name: DeleteProject :exec
delete from projects
where id = $1;
