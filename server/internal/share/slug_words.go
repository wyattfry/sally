package share

// shareSlugWords is a hand-curated list of 256 short, common,
// easy-to-spell English words used to build human-friendly share-link
// slugs.
// Selection criteria:
//   - 3–8 letters (so a 3-word slug stays short to type)
//   - one obvious spelling (no homophones, no British/American variants)
//   - phone-friendly (clear over voice, no ambiguous letter clusters)
//   - friendly/neutral connotations
//   - drawn from materials, nature, colors, landscape, household objects,
//     and architecture — feels at-home next to a project schedule
//
// Count must stay 256 to keep slug entropy predictable. When extending,
// append to the end and bump the count comment; do not reorder.
var shareSlugWords = []string{
	// woods (16)
	"oak", "pine", "maple", "cedar", "walnut", "birch", "elm", "teak",
	"cherry", "hickory", "spruce", "redwood", "bamboo", "cork", "willow", "alder",

	// stones & metals (16)
	"slate", "marble", "granite", "quartz", "brick", "stone", "steel", "brass",
	"copper", "bronze", "iron", "nickel", "chrome", "pewter", "silver", "gold",

	// colors (16)
	"amber", "ivory", "rose", "indigo", "scarlet", "olive", "cobalt", "teal",
	"crimson", "lilac", "coral", "ochre", "khaki", "ruby", "emerald", "pearl",

	// gems (8)
	"onyx", "topaz", "jade", "opal", "agate", "garnet", "lapis", "ember",

	// flora (16)
	"acorn", "leaf", "petal", "moss", "fern", "ivy", "vine", "bud",
	"seed", "root", "bark", "twig", "cone", "berry", "clover", "lily",

	// more flora & herbs (8)
	"tulip", "daisy", "iris", "poppy", "violet", "basil", "thyme", "mint",

	// friendly animals — birds & small mammals (16)
	"rabbit", "otter", "robin", "finch", "swan", "fox", "deer", "owl",
	"heron", "wren", "lark", "sparrow", "hawk", "falcon", "badger", "beaver",

	// more animals (16)
	"raccoon", "marten", "ferret", "moose", "elk", "stag", "fawn", "lynx",
	"hare", "mole", "newt", "frog", "toad", "trout", "salmon", "perch",

	// landscape (16)
	"meadow", "valley", "ridge", "creek", "cove", "harbor", "delta", "fjord",
	"glade", "grove", "marsh", "canyon", "mesa", "summit", "tundra", "prairie",

	// water (8)
	"atoll", "reef", "lagoon", "river", "brook", "stream", "glacier", "iceberg",

	// sky & weather (16)
	"comet", "cloud", "breeze", "thunder", "snow", "frost", "rain", "mist",
	"dew", "fog", "rainbow", "sunset", "dawn", "dusk", "twilight", "aurora",

	// household objects (16)
	"anchor", "lantern", "compass", "ladder", "kettle", "teapot", "lamp", "candle",
	"piano", "violin", "guitar", "flute", "drum", "trumpet", "harp", "cello",

	// desk & workshop (16)
	"book", "letter", "stamp", "ticket", "ribbon", "button", "thimble", "needle",
	"basket", "bucket", "broom", "shovel", "hammer", "wrench", "trowel", "ruler",

	// art tools (8)
	"pencil", "easel", "canvas", "palette", "brush", "chisel", "ink", "paper",

	// architecture (16)
	"arch", "dome", "spire", "gable", "eave", "porch", "patio", "studio",
	"loft", "atrium", "tower", "bridge", "tunnel", "barn", "cottage", "chalet",

	// buildings (8)
	"cabin", "manor", "lodge", "garden", "terrace", "pavilion", "gazebo", "balcony",

	// food (16)
	"fig", "plum", "pear", "peach", "lemon", "lime", "ginger", "cocoa",
	"butter", "sugar", "barley", "wheat", "kale", "oat", "corn", "sage",

	// textiles & cordage (16)
	"clay", "glass", "linen", "satin", "velvet", "wool", "cotton", "silk",
	"twine", "rope", "sail", "kite", "kayak", "canoe", "raft", "sled",

	// more landscape (8)
	"pond", "lake", "hill", "dune", "shore", "isle", "peak", "cliff",
}
