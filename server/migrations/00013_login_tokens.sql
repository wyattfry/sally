-- +goose Up
create table login_tokens (
    id         uuid primary key default gen_random_uuid(),
    user_id    uuid not null references users(id) on delete cascade,
    token_hash text not null unique,
    expires_at timestamptz not null default now() + interval '24 hours',
    used_at    timestamptz,
    created_at timestamptz not null default now()
);
