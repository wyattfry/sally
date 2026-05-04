-- +goose Up

-- Bump project.updated_at whenever a schedule is created, updated, or deleted.
-- +goose StatementBegin
create or replace function touch_project_on_schedule_change()
returns trigger language plpgsql as $$
begin
  update projects
  set updated_at = now()
  where id = coalesce(NEW.project_id, OLD.project_id);
  return null;
end;
$$;
-- +goose StatementEnd

create trigger schedules_touch_project
after insert or update or delete on schedules
for each row execute function touch_project_on_schedule_change();

-- Bump project.updated_at whenever a schedule item is created, updated, or deleted.
-- +goose StatementBegin
create or replace function touch_project_on_item_change()
returns trigger language plpgsql as $$
begin
  update projects
  set updated_at = now()
  from schedules
  where schedules.id = coalesce(NEW.schedule_id, OLD.schedule_id)
    and projects.id = schedules.project_id;
  return null;
end;
$$;
-- +goose StatementEnd

create trigger items_touch_project
after insert or update or delete on schedule_items
for each row execute function touch_project_on_item_change();

-- +goose Down
drop trigger if exists items_touch_project on schedule_items;
drop function if exists touch_project_on_item_change;
drop trigger if exists schedules_touch_project on schedules;
drop function if exists touch_project_on_schedule_change;
