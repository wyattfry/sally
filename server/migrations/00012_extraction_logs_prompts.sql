-- +goose Up
alter table extraction_logs
    add column prompt_text  text not null default '',
    add column response_text text not null default '';

-- +goose Down
alter table extraction_logs
    drop column prompt_text,
    drop column response_text;
