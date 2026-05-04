-- name: CreateProject :one
insert into projects (owner_user_id, name, address, description, thumbnail_url)
values ($1, $2, $3, $4, $5)
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
    description = $4,
    thumbnail_url = $5,
    updated_at = now()
where id = $1
returning *;

-- name: DeleteProject :exec
delete from projects
where id = $1;

-- name: GetProjectFirstItemImages :many
select si.source_image_url
from schedule_items si
join schedules s on s.id = si.schedule_id
where s.project_id = $1
  and si.source_image_url != ''
order by s.position, si.position, si.created_at
limit 4;
