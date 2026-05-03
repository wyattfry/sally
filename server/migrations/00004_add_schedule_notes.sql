-- +goose Up
alter table schedules add column notes text not null default '';

-- +goose Down
alter table schedules drop column notes;
