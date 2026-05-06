package schedcodes

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	queries "sally/server/internal/db/generated"
)

// ScheduleCodePrefix derives a single uppercase letter from the schedule name.
func ScheduleCodePrefix(name string) string {
	for _, r := range strings.ToUpper(strings.TrimSpace(name)) {
		if r >= 'A' && r <= 'Z' {
			return string(r)
		}
	}
	return "X"
}

func existingPrefix(items []queries.ScheduleItem) string {
	counts := map[string]int{}
	for _, item := range items {
		if len(item.Data) == 0 {
			continue
		}
		var data map[string]string
		if err := json.Unmarshal(item.Data, &data); err != nil {
			continue
		}
		code := strings.TrimSpace(data["code"])
		i := strings.LastIndex(code, "-")
		if i <= 0 {
			continue
		}
		if _, err := strconv.Atoi(code[i+1:]); err != nil {
			continue
		}
		counts[code[:i]]++
	}
	best, bestCount := "", 0
	for prefix, count := range counts {
		if count > bestCount || (count == bestCount && prefix < best) {
			best, bestCount = prefix, count
		}
	}
	return best
}

// NextCode returns the next sequential code for a schedule.
func NextCode(items []queries.ScheduleItem, scheduleName string) string {
	prefix := existingPrefix(items)
	if prefix == "" {
		prefix = ScheduleCodePrefix(scheduleName)
	}

	max := 0
	needle := prefix + "-"
	for _, item := range items {
		if len(item.Data) == 0 {
			continue
		}
		var data map[string]string
		if err := json.Unmarshal(item.Data, &data); err != nil {
			continue
		}
		code := data["code"]
		if !strings.HasPrefix(code, needle) {
			continue
		}
		n, err := strconv.Atoi(code[len(needle):])
		if err != nil {
			continue
		}
		if n > max {
			max = n
		}
	}
	return fmt.Sprintf("%s-%d", prefix, max+1)
}
