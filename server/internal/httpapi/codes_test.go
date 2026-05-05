package httpapi

import (
	"encoding/json"
	"testing"

	queries "sally/server/internal/db/generated"
)

func TestScheduleCodePrefix(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"Paint", "PA"},
		{"Appliance", "AP"},
		{"Door", "DO"},
		{"Window", "WI"},
		{"General", "GE"},
		{"Insulation", "IN"},
		{"Specialties", "SP"},
		{"Electrical Fixture", "EF"},
		{"Door Hardware", "DH"},
		{"Door Hardware Schedule", "DHS"},
		{"", "X"},
		{"A", "A"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := scheduleCodePrefix(tc.name)
			if got != tc.want {
				t.Errorf("scheduleCodePrefix(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestNextCode(t *testing.T) {
	makeItem := func(code string) queries.ScheduleItem {
		data, _ := json.Marshal(map[string]string{"code": code})
		return queries.ScheduleItem{Data: data}
	}

	t.Run("empty schedule starts at 1", func(t *testing.T) {
		got := nextCode(nil, "PA")
		if got != "PA-1" {
			t.Errorf("got %q, want PA-1", got)
		}
	})

	t.Run("increments past existing", func(t *testing.T) {
		items := []queries.ScheduleItem{
			makeItem("PA-1"),
			makeItem("PA-2"),
			makeItem("PA-3"),
		}
		got := nextCode(items, "PA")
		if got != "PA-4" {
			t.Errorf("got %q, want PA-4", got)
		}
	})

	t.Run("ignores different prefix", func(t *testing.T) {
		items := []queries.ScheduleItem{
			makeItem("DO-1"),
			makeItem("DO-2"),
		}
		got := nextCode(items, "PA")
		if got != "PA-1" {
			t.Errorf("got %q, want PA-1", got)
		}
	})

	t.Run("ignores non-numeric suffix", func(t *testing.T) {
		items := []queries.ScheduleItem{
			makeItem("PA-1"),
			makeItem("PA-abc"),
			makeItem("PA-2"),
		}
		got := nextCode(items, "PA")
		if got != "PA-3" {
			t.Errorf("got %q, want PA-3", got)
		}
	})

	t.Run("ignores empty data", func(t *testing.T) {
		items := []queries.ScheduleItem{
			{Data: nil},
			makeItem("PA-5"),
		}
		got := nextCode(items, "PA")
		if got != "PA-6" {
			t.Errorf("got %q, want PA-6", got)
		}
	})
}
