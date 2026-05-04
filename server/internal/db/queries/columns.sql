-- name: CreateScheduleColumn :one
insert into schedule_columns (schedule_id, key, label, kind, position)
values ($1, $2, $3, $4, $5)
returning *;

-- name: ListScheduleColumns :many
select *
from schedule_columns
where schedule_id = $1
order by position asc, created_at asc;

-- name: DeleteScheduleColumn :exec
delete from schedule_columns
where id = $1;

-- name: DeleteScheduleColumnsBySchedule :exec
delete from schedule_columns
where schedule_id = $1;
