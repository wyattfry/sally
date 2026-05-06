-- +goose Up
create table project_members (
    id uuid primary key default gen_random_uuid(),
    project_id uuid not null references projects(id) on delete cascade,
    user_id uuid not null references users(id) on delete cascade,
    invited_by_user_id uuid not null references users(id),
    created_at timestamptz not null default now(),
    unique(project_id, user_id)
);

create index project_members_user_id_idx on project_members(user_id);
create index project_members_project_id_idx on project_members(project_id);

-- +goose Down
drop table if exists project_members;
