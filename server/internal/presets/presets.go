package presets

// ColumnDef describes a single column in a schedule preset.
type ColumnDef struct {
	Key      string
	Label    string
	Position int32
}

// Schedules maps preset names to their default column sets.
// These are applied when a new schedule is created; users can add/remove
// columns after the fact.
var Schedules = map[string][]ColumnDef{
	"general": {
		{Key: "zone", Label: "Zone", Position: 1},
		{Key: "code", Label: "Code", Position: 2},
		{Key: "manufacturer", Label: "Manufacturer", Position: 3},
		{Key: "model_number", Label: "Model", Position: 4},
		{Key: "finish", Label: "Finish", Position: 5},
		{Key: "notes", Label: "Notes", Position: 6},
	},
	"appliance": {
		{Key: "code", Label: "Code", Position: 1},
		{Key: "description", Label: "Description", Position: 2},
		{Key: "product_info", Label: "Product Information", Position: 3},
		{Key: "notes", Label: "Notes", Position: 4},
	},
	"window": {
		{Key: "code", Label: "Code", Position: 1},
		{Key: "description", Label: "Description", Position: 2},
		{Key: "model_number", Label: "Model Number", Position: 3},
		{Key: "rough_opening", Label: "Rough Opening", Position: 4},
		{Key: "overall_jamb", Label: "Overall Jamb", Position: 5},
		{Key: "swing", Label: "Swing", Position: 6},
	},
	"door": {
		{Key: "code", Label: "Code", Position: 1},
		{Key: "description", Label: "Description", Position: 2},
		{Key: "model_number", Label: "Model Number", Position: 3},
		{Key: "notes", Label: "Notes", Position: 4},
	},
	"door_hardware": {
		{Key: "type", Label: "Type", Position: 1},
		{Key: "description", Label: "Description", Position: 2},
		{Key: "mfg_number", Label: "MFG #", Position: 3},
		{Key: "finish", Label: "Finish", Position: 4},
	},
	"electrical_fixture": {
		{Key: "code", Label: "Code", Position: 1},
		{Key: "description", Label: "Description", Position: 2},
		{Key: "manf_info", Label: "Manufacturer Info", Position: 3},
		{Key: "finish_notes", Label: "Finish / Notes", Position: 4},
	},
	"paint": {
		{Key: "code", Label: "Code", Position: 1},
		{Key: "color", Label: "Color", Position: 2},
		{Key: "manufacturer", Label: "Manufacturer", Position: 3},
		{Key: "notes", Label: "Notes", Position: 4},
	},
	"insulation": {
		{Key: "description", Label: "Description", Position: 1},
		{Key: "r_value", Label: "R-Value", Position: 2},
		{Key: "notes", Label: "Notes", Position: 3},
	},
	"specialties": {
		{Key: "code", Label: "Code", Position: 1},
		{Key: "description", Label: "Description", Position: 2},
	},
}

// Default returns the column set used when no specific preset is chosen.
func Default() []ColumnDef {
	return Schedules["general"]
}

// Get returns the named preset, falling back to Default.
func Get(name string) []ColumnDef {
	if cols, ok := Schedules[name]; ok {
		return cols
	}
	return Default()
}
