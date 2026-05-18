package presets

import "testing"

func TestInferByName(t *testing.T) {
	cases := []struct {
		name   string
		want   string
		wantOK bool
	}{
		{"Paint", "paint", true},
		{"Paint Schedule", "paint", true},
		{"Interior Paint", "paint", true},
		{"Appliance Schedule", "appliance", true},
		{"Door Hardware", "door_hardware", true},  // longer match wins over "door"
		{"Doors", "door", true},
		{"Lighting Schedule", "electrical_fixture", true},
		{"Electrical", "electrical_fixture", true},
		{"Miscellaneous", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := InferByName(tc.name)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("InferByName(%q) = (%q, %v), want (%q, %v)",
					tc.name, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestPaintPresetHasColorAndFinish(t *testing.T) {
	cols := Get("paint")
	keys := map[string]bool{}
	for _, c := range cols {
		keys[c.Key] = true
	}
	for _, want := range []string{"color", "finish"} {
		if !keys[want] {
			t.Errorf("paint preset missing column %q (got %v)", want, keys)
		}
	}
}
