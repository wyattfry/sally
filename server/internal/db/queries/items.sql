-- name: CreateScheduleItem :one
insert into schedule_items (
    schedule_id,
    data,
    zone,
    source_url,
    source_title,
    source_image_url,
    source_pdf_links,
    position
)
values ($1, $2, $3, $4, $5, $6, $7, $8)
returning *;

-- name: GetScheduleItem :one
select *
from schedule_items
where id = $1;

-- name: ListScheduleItems :many
select *
from schedule_items
where schedule_id = $1
order by zone asc, position asc, created_at asc;

-- name: UpdateScheduleItem :one
update schedule_items
set data             = $2,
    zone             = $3,
    source_url       = $4,
    source_title     = $5,
    source_image_url = $6,
    source_pdf_links = $7,
    position         = $8,
    updated_at       = now()
where id = $1
returning *;

-- name: DeleteScheduleItem :exec
delete from schedule_items
where id = $1;
