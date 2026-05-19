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

// Canonical keys are what the LLM extractor produces — manufacturer,
// model_number, finish, notes. Presets that name these columns differently
// (e.g. "manf_info" / "finish_notes") will receive extraction data that
// doesn't map to any visible column, so the new-schedule flow renders
// empty fields. Guard the canonical set across presets that should carry
// these fields.
func TestPresetsUseCanonicalKeysForCommonFields(t *testing.T) {
	cases := []struct {
		preset       string
		mustInclude  []string
		mustNotMatch []string // legacy bundled keys we don't want re-introduced
	}{
		{"appliance", []string{"manufacturer", "model_number"}, []string{"product_info"}},
		{"electrical_fixture", []string{"manufacturer", "model_number", "finish"}, []string{"manf_info", "finish_notes"}},
		{"door_hardware", []string{"manufacturer", "model_number"}, []string{"mfg_number"}},
		{"door", []string{"manufacturer", "model_number"}, nil},
		{"window", []string{"manufacturer", "model_number"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.preset, func(t *testing.T) {
			cols := Get(tc.preset)
			keys := map[string]bool{}
			for _, c := range cols {
				keys[c.Key] = true
			}
			for _, want := range tc.mustInclude {
				if !keys[want] {
					t.Errorf("%q preset missing canonical key %q (got %v)", tc.preset, want, keys)
				}
			}
			for _, banned := range tc.mustNotMatch {
				if keys[banned] {
					t.Errorf("%q preset still uses bundled/legacy key %q — should be on canonical keys instead", tc.preset, banned)
				}
			}
		})
	}
}
