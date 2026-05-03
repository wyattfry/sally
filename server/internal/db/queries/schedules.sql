-- name: CreateSchedule :one
insert into schedules (project_id, name, position)
values ($1, $2, $3)
returning *;

-- name: GetSchedule :one
select *
from schedules
where id = $1;

-- name: ListSchedulesByProject :many
select *
from schedules
where project_id = $1
order by position asc, created_at asc;

-- name: UpdateSchedule :one
update schedules
set name = $2,
    position = $3,
    updated_at = now()
where id = $1
returning *;

-- name: DeleteSchedule :exec
delete from schedules
where id = $1;
