-- +goose Up
alter table schedule_items
    add column source_image_urls text[] not null default '{}';

-- +goose Down
alter table schedule_items drop column source_image_urls;
