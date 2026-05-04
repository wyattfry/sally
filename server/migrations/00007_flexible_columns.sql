-- +goose Up

-- Allow schedules to be either a table of items or a free-form note block.
alter table schedules
    add column kind text not null default 'items';

-- Column definitions owned by each schedule.
create table schedule_columns (
    id          uuid primary key default gen_random_uuid(),
    schedule_id uuid not null references schedules(id) on delete cascade,
    key         text not null,
    label       text not null,
    kind        text not null default 'text',
    position    integer not null default 0,
    created_at  timestamptz not null default now()
);
create unique index schedule_columns_schedule_key_idx on schedule_columns(schedule_id, key);
create index schedule_columns_schedule_position_idx on schedule_columns(schedule_id, position, created_at);

-- Replace fixed item columns with a flexible jsonb bag.
alter table schedule_items add column data jsonb not null default '{}';

-- Migrate existing fixed-column data into the jsonb bag, skipping empty values.
-- +goose StatementBegin
update schedule_items set data = jsonb_strip_nulls(jsonb_build_object(
    'code',               nullif(trim(code), ''),
    'title',              nullif(trim(title), ''),
    'description',        nullif(trim(description), ''),
    'manufacturer',       nullif(trim(manufacturer), ''),
    'model_number',       nullif(trim(model_number), ''),
    'finish',             nullif(trim(finish), ''),
    'finish_model_number', nullif(trim(finish_model_number), ''),
    'notes',              nullif(trim(notes), '')
));
-- +goose StatementEnd

-- Seed schedule_columns for every existing items-type schedule.
-- +goose StatementBegin
insert into schedule_columns (schedule_id, key, label, kind, position)
select s.id, v.key, v.label, 'text', v.pos
from schedules s
cross join (values
    (1, 'code',                'Code'),
    (2, 'title',               'Title'),
    (3, 'description',         'Description'),
    (4, 'manufacturer',        'Manufacturer'),
    (5, 'model_number',        'Model Number'),
    (6, 'finish',              'Finish'),
    (7, 'finish_model_number', 'Finish Model #'),
    (8, 'notes',               'Notes')
) as v(pos, key, label)
where s.kind = 'items';
-- +goose StatementEnd

-- Drop the now-migrated fixed columns.
alter table schedule_items
    drop column code,
    drop column title,
    drop column description,
    drop column manufacturer,
    drop column model_number,
    drop column finish,
    drop column finish_model_number,
    drop column notes;

-- +goose Down

alter table schedule_items
    add column code                text not null default '',
    add column title               text not null default '',
    add column description         text not null default '',
    add column manufacturer        text not null default '',
    add column model_number        text not null default '',
    add column finish              text not null default '',
    add column finish_model_number text not null default '',
    add column notes               text not null default '';

-- +goose StatementBegin
update schedule_items set
    code                = coalesce(data->>'code', ''),
    title               = coalesce(data->>'title', ''),
    description         = coalesce(data->>'description', ''),
    manufacturer        = coalesce(data->>'manufacturer', ''),
    model_number        = coalesce(data->>'model_number', ''),
    finish              = coalesce(data->>'finish', ''),
    finish_model_number = coalesce(data->>'finish_model_number', ''),
    notes               = coalesce(data->>'notes', '');
-- +goose StatementEnd

drop table if exists schedule_columns;
alter table schedules drop column if exists kind;
alter table schedule_items drop column if exists data;
