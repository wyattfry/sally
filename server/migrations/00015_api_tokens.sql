-- +goose Up
create table api_tokens (
    id           uuid primary key default gen_random_uuid(),
    user_id      uuid not null references users(id) on delete cascade,
    label        text not null default '',
    token_hash   text not null unique,
    created_at   timestamptz not null default now(),
    last_used_at timestamptz
);

-- +goose Down
drop table api_tokens;
