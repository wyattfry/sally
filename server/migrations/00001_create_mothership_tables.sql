-- +goose Up
create extension if not exists pgcrypto;

create table users (
    id uuid primary key default gen_random_uuid(),
    email text not null unique,
    name text not null default '',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create table projects (
    id uuid primary key default gen_random_uuid(),
    owner_user_id uuid not null references users(id) on delete cascade,
    name text not null,
    address text not null default '',
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index projects_owner_user_id_idx on projects(owner_user_id);

create table schedules (
    id uuid primary key default gen_random_uuid(),
    project_id uuid not null references projects(id) on delete cascade,
    name text not null,
    position integer not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index schedules_project_id_position_idx on schedules(project_id, position, created_at);

create table schedule_items (
    id uuid primary key default gen_random_uuid(),
    schedule_id uuid not null references schedules(id) on delete cascade,
    code text not null default '',
    title text not null default '',
    description text not null default '',
    manufacturer text not null default '',
    model_number text not null default '',
    finish text not null default '',
    finish_model_number text not null default '',
    notes text not null default '',
    source_url text not null default '',
    source_title text not null default '',
    source_image_url text not null default '',
    source_pdf_links text[] not null default '{}',
    position integer not null default 0,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now()
);

create index schedule_items_schedule_id_position_idx on schedule_items(schedule_id, position, created_at);

create table project_share_links (
    id uuid primary key default gen_random_uuid(),
    project_id uuid not null references projects(id) on delete cascade,
    token_hash text not null unique,
    label text not null default '',
    active boolean not null default true,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    last_viewed_at timestamptz
);

create index project_share_links_project_id_idx on project_share_links(project_id);
create index project_share_links_active_token_hash_idx on project_share_links(token_hash) where active;

-- +goose Down
drop table if exists project_share_links;
drop table if exists schedule_items;
drop table if exists schedules;
drop table if exists projects;
drop table if exists users;
