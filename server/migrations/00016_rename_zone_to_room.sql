-- +goose Up
alter table schedule_items rename column zone to room;

update schedule_columns set key = 'room' where key = 'zone';
update schedule_columns set label = 'Room' where label = 'Zone';

update schedule_items
  set data = (data - 'zone') || jsonb_build_object('room', data->>'zone')
  where data ? 'zone';

-- +goose Down
alter table schedule_items rename column room to zone;

update schedule_columns set key = 'zone' where key = 'room';
update schedule_columns set label = 'Zone' where label = 'Room';

update schedule_items
  set data = (data - 'room') || jsonb_build_object('zone', data->>'room')
  where data ? 'room';
