-- +goose Up
alter table projects add column description text not null default '';
alter table projects add column thumbnail_url text not null default '';

-- +goose Down
alter table projects drop column thumbnail_url;
alter table projects drop column description;
