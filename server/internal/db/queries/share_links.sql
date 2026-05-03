-- name: CreateProjectShareLink :one
insert into project_share_links (project_id, token_hash, token, label)
values ($1, $2, $3, $4)
returning *;

-- name: ListProjectShareLinks :many
select *
from project_share_links
where project_id = $1
order by created_at desc;

-- name: GetActiveProjectShareLinkByHash :one
select *
from project_share_links
where token_hash = $1 and active = true;

-- name: DeactivateProjectShareLink :one
update project_share_links
set active = false,
    updated_at = now()
where id = $1
returning *;

-- name: MarkProjectShareLinkViewed :exec
update project_share_links
set last_viewed_at = now()
where id = $1;

-- name: GetActiveProjectShareLinkByProject :one
select *
from project_share_links
where project_id = $1 and active = true
order by created_at desc
limit 1;

-- name: DeactivateProjectShareLinks :exec
update project_share_links
set active = false,
    updated_at = now()
where project_id = $1 and active = true;
