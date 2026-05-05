package httpapi

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	queries "sally/server/internal/db/generated"
)

// scheduleCodePrefix derives a short uppercase prefix from a schedule name.
// Single word  → first two letters  ("Paint" → "PA", "Door" → "DO").
// Multi-word   → initials, up to 3  ("Door Hardware" → "DH", "Electrical Fixture Schedule" → "EFS").
func scheduleCodePrefix(name string) string {
	words := strings.Fields(name)
	if len(words) == 0 {
		return "X"
	}
	if len(words) == 1 {
		w := strings.ToUpper(words[0])
		if len(w) >= 2 {
			return w[:2]
		}
		return w
	}
	var sb strings.Builder
	for i, w := range words {
		if i >= 3 {
			break
		}
		uw := strings.ToUpper(w)
		if len(uw) > 0 {
			sb.WriteByte(uw[0])
		}
	}
	return sb.String()
}

// nextCode returns the next sequential code for prefix by scanning existing
// items. It finds the highest numeric suffix already in use (e.g. "PA-3")
// and returns prefix-N+1. Returns prefix-1 when no matching codes exist.
func nextCode(items []queries.ScheduleItem, prefix string) string {
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
