-- +goose Up
alter table schedule_items add column zone text not null default '';

-- +goose Down
alter table schedule_items drop column zone;
