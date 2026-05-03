-- +goose Up
alter table project_share_links add column token text not null default '';

-- +goose Down
alter table project_share_links drop column token;
