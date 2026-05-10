-- name: AddProjectMember :exec
insert into project_members (project_id, user_id, invited_by_user_id)
values ($1, $2, $3)
on conflict (project_id, user_id) do nothing;

-- name: RemoveProjectMember :exec
delete from project_members
where project_id = $1 and user_id = $2;

-- name: GetProjectMember :one
select id, project_id, user_id, invited_by_user_id, created_at
from project_members
where project_id = $1 and user_id = $2;

-- name: ListProjectMembersWithUser :many
select pm.id, pm.project_id, pm.user_id, pm.invited_by_user_id, pm.created_at,
       u.email as user_email, u.name as user_name
from project_members pm
join users u on u.id = pm.user_id
where pm.project_id = $1
order by pm.created_at;

-- name: ListSharedProjects :many
select p.id, p.owner_user_id, p.name, p.address, p.created_at, p.updated_at, p.description, p.thumbnail_url
from projects p
join project_members pm on pm.project_id = p.id
where pm.user_id = $1
order by p.updated_at desc, p.created_at desc;

-- name: ListSharedProjectsWithOwner :many
select p.id, p.owner_user_id, p.name, p.address, p.created_at, p.updated_at, p.description, p.thumbnail_url,
       coalesce(nullif(u.name, ''), u.email, p.owner_user_id::text) as owner_display_name
from projects p
join project_members pm on pm.project_id = p.id
join users u on u.id = p.owner_user_id
where pm.user_id = $1
order by p.updated_at desc, p.created_at desc;
