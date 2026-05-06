-- +goose Up
create table extraction_logs (
    id            uuid        primary key default gen_random_uuid(),
    request_id    text        not null,
    schedule_id   uuid        references schedules(id) on delete set null,
    provider      text        not null default '',
    model         text        not null default '',
    prompt_version text       not null default '',
    duration_ms   integer     not null default 0,
    success       boolean     not null default true,
    error_message text        not null default '',
    page_url      text        not null default '',
    created_at    timestamptz not null default now()
);

create index extraction_logs_created_at_idx  on extraction_logs(created_at desc);
create index extraction_logs_schedule_id_idx on extraction_logs(schedule_id);

-- +goose Down
drop table if exists extraction_logs;
