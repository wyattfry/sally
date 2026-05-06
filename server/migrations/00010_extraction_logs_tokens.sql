-- +goose Up
alter table extraction_logs
    add column if not exists prompt_tokens     integer not null default 0,
    add column if not exists completion_tokens integer not null default 0,
    add column if not exists missing_fields_count integer not null default 0;

-- +goose Down
alter table extraction_logs
    drop column if exists prompt_tokens,
    drop column if exists completion_tokens,
    drop column if exists missing_fields_count;
