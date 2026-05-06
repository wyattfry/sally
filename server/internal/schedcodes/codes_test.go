package schedcodes

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
		{"Paint", "P"},
		{"Appliance", "A"},
		{"Appliance Schedule", "A"},
		{"Door", "D"},
		{"Window", "W"},
		{"General", "G"},
		{"Insulation", "I"},
		{"Specialties", "S"},
		{"Electrical Fixture", "E"},
		{"Door Hardware", "D"},
		{"Door Hardware Schedule", "D"},
		{"", "X"},
		{"123 Paint", "P"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScheduleCodePrefix(tc.name)
			if got != tc.want {
				t.Errorf("ScheduleCodePrefix(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

func TestExistingPrefix(t *testing.T) {
	makeItem := func(code string) queries.ScheduleItem {
		data, _ := json.Marshal(map[string]string{"code": code})
		return queries.ScheduleItem{Data: data}
	}

	t.Run("no items returns empty", func(t *testing.T) {
		if got := existingPrefix(nil); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})

	t.Run("detects single-letter prefix", func(t *testing.T) {
		items := []queries.ScheduleItem{makeItem("A-1"), makeItem("A-2"), makeItem("A-3")}
		if got := existingPrefix(items); got != "A" {
			t.Errorf("got %q, want A", got)
		}
	})

	t.Run("detects multi-char prefix", func(t *testing.T) {
		items := []queries.ScheduleItem{makeItem("EF-1"), makeItem("EF-2")}
		if got := existingPrefix(items); got != "EF" {
			t.Errorf("got %q, want EF", got)
		}
	})

	t.Run("ignores codes without numeric suffix", func(t *testing.T) {
		items := []queries.ScheduleItem{makeItem("A-1"), makeItem("A-two"), makeItem("A-3")}
		if got := existingPrefix(items); got != "A" {
			t.Errorf("got %q, want A", got)
		}
	})

	t.Run("returns most common when mixed", func(t *testing.T) {
		items := []queries.ScheduleItem{
			makeItem("A-1"), makeItem("A-2"), makeItem("A-3"),
			makeItem("B-1"),
		}
		if got := existingPrefix(items); got != "A" {
			t.Errorf("got %q, want A", got)
		}
	})
}

func TestNextCode(t *testing.T) {
	makeItem := func(code string) queries.ScheduleItem {
		data, _ := json.Marshal(map[string]string{"code": code})
		return queries.ScheduleItem{Data: data}
	}

	t.Run("empty schedule derives from name", func(t *testing.T) {
		got := NextCode(nil, "Appliance Schedule")
		if got != "A-1" {
			t.Errorf("got %q, want A-1", got)
		}
	})

	t.Run("follows existing prefix not schedule name", func(t *testing.T) {
		items := []queries.ScheduleItem{makeItem("AP-1"), makeItem("AP-2")}
		got := NextCode(items, "Appliance Schedule")
		if got != "AP-3" {
			t.Errorf("got %q, want AP-3", got)
		}
	})

	t.Run("increments past max not count", func(t *testing.T) {
		items := []queries.ScheduleItem{makeItem("A-1"), makeItem("A-3")}
		got := NextCode(items, "Appliance")
		if got != "A-4" {
			t.Errorf("got %q, want A-4", got)
		}
	})

	t.Run("follows established prefix over schedule name", func(t *testing.T) {
		items := []queries.ScheduleItem{makeItem("D-1"), makeItem("D-2")}
		got := NextCode(items, "Paint")
		if got != "D-3" {
			t.Errorf("got %q, want D-3", got)
		}
	})

	t.Run("ignores nil data", func(t *testing.T) {
		items := []queries.ScheduleItem{{Data: nil}, makeItem("A-5")}
		got := NextCode(items, "Appliance")
		if got != "A-6" {
			t.Errorf("got %q, want A-6", got)
		}
	})
}
