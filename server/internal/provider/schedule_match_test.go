package provider

import (
	"testing"

	"sally/server/internal/extract"
)

func TestSnapSuggestedScheduleName(t *testing.T) {
	cases := []struct {
		name      string
		suggested string
		existing  []string
		want      string
		wantOK    bool
	}{
		{"exact match", "Paint", []string{"Paint", "Lighting"}, "Paint", true},
		{"case insensitive", "PAINT", []string{"Paint"}, "Paint", true},
		{"strip schedule suffix on suggestion", "Lighting Schedule", []string{"Lighting"}, "Lighting", true},
		{"strip schedule suffix on existing", "Lighting", []string{"Lighting Schedule"}, "Lighting Schedule", true},
		{"suggestion superset snaps", "Plumbing Fixtures", []string{"Plumbing"}, "Plumbing", true},
		{"existing superset snaps", "Bath", []string{"Bath Fixtures"}, "Bath Fixtures", true},
		{"no match returns false", "Lighting", []string{"Paint", "Plumbing"}, "", false},
		{"empty suggestion returns false", "", []string{"Paint"}, "", false},
		{"placeholder existing doesn't capture", "Plumbing", []string{"New Schedule"}, "", false},
		{"unrelated tokens don't match", "Wall Sconce", []string{"Paint"}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SnapSuggestedScheduleName(tc.suggested, tc.existing)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("SnapSuggestedScheduleName(%q, %v) = (%q, %v), want (%q, %v)",
					tc.suggested, tc.existing, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestFilterPromptableSchedules(t *testing.T) {
	in := []extract.ScheduleSummary{
		{Name: "Paint", Rooms: []string{"Kitchen"}},
		{Name: "New Schedule", Rooms: nil},
		{Name: "New Schedule 2", Rooms: nil},
		{Name: "New Schedule", Rooms: []string{"Bath"}}, // has items — keep
		{Name: "Real Schedule", Rooms: nil},             // not placeholder pattern — keep
		{Name: "New Note 3", Rooms: nil},
	}
	out := filterPromptableSchedules(in)
	wantNames := []string{"Paint", "New Schedule", "Real Schedule"}
	if len(out) != len(wantNames) {
		t.Fatalf("got %d schedules, want %d: %+v", len(out), len(wantNames), out)
	}
	for i, w := range wantNames {
		if out[i].Name != w {
			t.Errorf("out[%d].Name = %q, want %q", i, out[i].Name, w)
		}
	}
}
