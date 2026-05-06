package httpapi

import (
	queries "sally/server/internal/db/generated"
	"sally/server/internal/schedcodes"
)

func nextCode(items []queries.ScheduleItem, scheduleName string) string {
	return schedcodes.NextCode(items, scheduleName)
}
