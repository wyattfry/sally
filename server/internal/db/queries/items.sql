-- name: CreateScheduleItem :one
insert into schedule_items (
    schedule_id,
    code,
    title,
    description,
    manufacturer,
    model_number,
    finish,
    finish_model_number,
    notes,
    source_url,
    source_title,
    source_image_url,
    source_pdf_links,
    position
)
values (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13, $14
)
returning *;

-- name: GetScheduleItem :one
select *
from schedule_items
where id = $1;

-- name: ListScheduleItems :many
select *
from schedule_items
where schedule_id = $1
order by position asc, created_at asc;

-- name: UpdateScheduleItem :one
update schedule_items
set code = $2,
    title = $3,
    description = $4,
    manufacturer = $5,
    model_number = $6,
    finish = $7,
    finish_model_number = $8,
    notes = $9,
    source_url = $10,
    source_title = $11,
    source_image_url = $12,
    source_pdf_links = $13,
    position = $14,
    updated_at = now()
where id = $1
returning *;

-- name: DeleteScheduleItem :exec
delete from schedule_items
where id = $1;
