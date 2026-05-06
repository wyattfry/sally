-- +goose Up

-- Drop retired columns from all schedules (title, description, finish_model_number)
delete from schedule_columns
where key in ('title', 'description', 'finish_model_number');

-- Rename "Model Number" → "Model"
update schedule_columns
set label = 'Model'
where key = 'model_number' and label = 'Model Number';

-- Resequence positions to close gaps left by deletions
with ranked as (
    select id,
           row_number() over (partition by schedule_id order by position) as new_pos
    from schedule_columns
)
update schedule_columns sc
set position = r.new_pos
from ranked r
where sc.id = r.id;

-- +goose Down

-- Cannot restore deleted column definitions (data is retained in item JSONB)
select 1;
