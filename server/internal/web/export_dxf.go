package web

// DXF schedule export — AC1009 (AutoCAD R12) ASCII format.
//
// R12 is chosen for maximum compatibility: every CAD package that can read
// DXF at all reads R12. The trade-off is no MTEXT, so multi-line notes are
// split into multiple TEXT entities at cell boundary.
//
// Coordinate system: inches, Y increases upward. Each schedule is laid out
// as a table block stacked vertically with a gap between them.
//
// Required DXF sections (many parsers reject the file if any are absent):
//   HEADER → TABLES (LTYPE + LAYER) → BLOCKS ($MODEL_SPACE stub) → ENTITIES → EOF

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
)

// ── layout constants (inches) ────────────────────────────────────────────────

const (
	dxfMarginX    = 0.5   // left/right page margin
	dxfMarginY    = 0.5   // bottom margin
	dxfGapY       = 0.75  // vertical gap between consecutive schedule tables
	dxfTitleH     = 0.50  // title row height
	dxfHeaderH    = 0.32  // column-label row height
	dxfRowH       = 0.28  // data row height
	dxfTextTitle  = 0.18  // title text height
	dxfTextHeader = 0.09  // column-label text height
	dxfTextData   = 0.08  // cell text height
	dxfPad        = 0.07  // horizontal padding inside a cell
	dxfTextBaseFr = 0.28  // fraction of row height for text baseline (from bottom)
	dxfMinColW    = 0.90  // minimum column width
	dxfMaxColW    = 4.50  // normal maximum column width
	dxfNotesMaxW  = 5.50  // notes/description column maximum
	dxfCharW      = 0.055 // estimated character width at dxfTextData height
)

// ── handler ─────────────────────────────────────────────────────────────────

func (a app) exportProjectDXF(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProject(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	schedules, err := a.schedulesWithItems(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	// Filter to item schedules only.
	var tables []scheduleWithItems
	for _, sw := range schedules {
		if sw.Schedule.Kind == "items" {
			tables = append(tables, sw)
		}
	}
	if len(tables) == 0 {
		http.Error(w, "no item schedules to export", http.StatusBadRequest)
		return
	}

	filename := strings.NewReplacer(" ", "_", "/", "-").Replace(project.Name) + ".dxf"
	w.Header().Set("Content-Type", "application/dxf")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	dxfWriteProject(w, project.Name, tables)
}

// ── top-level writer ────────────────────────────────────────────────────────

func dxfWriteProject(w io.Writer, projectName string, tables []scheduleWithItems) {
	d := &dxfWriter{w: w}

	// Compute total drawing height to set LIMMAX.
	totalH := dxfMarginY
	for i, sw := range tables {
		rows := dxfCountRows(sw)
		totalH += dxfTitleH + dxfHeaderH + float64(rows)*dxfRowH
		if i < len(tables)-1 {
			totalH += dxfGapY
		}
	}
	totalH += dxfMarginY
	totalW := dxfMarginX + dxfMaxColW*7 + dxfMarginX // generous estimate

	d.header(totalW, totalH)
	d.tables()
	d.blocks()
	d.sectionStart("ENTITIES")

	// Draw each schedule table, stacked top-to-bottom.
	curY := totalH - dxfMarginY // current top-of-table Y
	for _, sw := range tables {
		curY = dxfDrawTable(d, sw, curY)
		curY -= dxfGapY
	}

	d.sectionEnd()
	d.eof()
}

// dxfCountRows returns the total number of data rows across all room groups.
func dxfCountRows(sw scheduleWithItems) int {
	n := 0
	for _, g := range sw.Groups {
		n += len(g.Items)
	}
	return n
}

// dxfRow holds one item's flat display values.
type dxfRow struct {
	room    string
	dataMap map[string]string
}

// dxfDrawTable draws a single schedule table starting with its top edge at
// topY and returns the Y coordinate of the table's bottom edge.
func dxfDrawTable(d *dxfWriter, sw scheduleWithItems, topY float64) float64 {
	// Flatten items.
	var rows []dxfRow
	for _, g := range sw.Groups {
		for _, it := range g.Items {
			rows = append(rows, dxfRow{room: it.Room, dataMap: it.DataMap})
		}
	}

	// Build columns.
	type col struct {
		key, label string
		width      float64
		isNotes    bool
	}
	var cols []col
	hasRoom := false
	for _, r2 := range rows {
		if r2.room != "" {
			hasRoom = true
			break
		}
	}
	if hasRoom {
		w2 := dxfColWidthFromValues("Room", func(i int) string { return rows[i].room }, len(rows), false)
		cols = append(cols, col{key: "room", label: "Room", width: w2})
	}
	for _, sc := range sw.Columns {
		if sc.Key == "room" {
			continue
		}
		notes := sc.Key == "notes" || sc.Key == "description"
		w2 := dxfColWidthFromValues(sc.Label, func(i int) string { return rows[i].dataMap[sc.Key] }, len(rows), notes)
		cols = append(cols, col{key: sc.Key, label: sc.Label, width: w2, isNotes: notes})
	}
	if len(cols) == 0 {
		return topY
	}

	tableW := 0.0
	for _, c := range cols {
		tableW += c.width
	}
	x0 := dxfMarginX // left edge of table

	// ── Title row ─────────────────────────────────────────────────────────
	yTop := topY
	yBottom := yTop - dxfTitleH
	d.rect("BORDER", x0, yBottom, x0+tableW, yTop)
	d.text("TITLE", x0+dxfPad, yBottom+dxfTitleH*dxfTextBaseFr, dxfTextTitle, sw.Schedule.Name)

	// ── Column header row ─────────────────────────────────────────────────
	y2 := yBottom
	hdrBottom := y2 - dxfHeaderH
	d.hline("BORDER", x0, x0+tableW, hdrBottom)
	d.vline("BORDER", x0, hdrBottom, y2)
	d.vline("BORDER", x0+tableW, hdrBottom, y2)
	x := x0
	for i, c := range cols {
		d.text("COLHEAD", x+dxfPad, hdrBottom+dxfHeaderH*dxfTextBaseFr, dxfTextHeader, c.label)
		if i < len(cols)-1 {
			d.vline("BORDER", x+c.width, hdrBottom, y2)
		}
		x += c.width
	}

	// ── Data rows ─────────────────────────────────────────────────────────
	for ri, row2 := range rows {
		y2 = hdrBottom - float64(ri)*dxfRowH
		rBottom := y2 - dxfRowH
		d.hline("BORDER", x0, x0+tableW, rBottom)
		d.vline("BORDER", x0, rBottom, y2)
		d.vline("BORDER", x0+tableW, rBottom, y2)
		x = x0
		for ci, c := range cols {
			var val string
			if c.key == "room" {
				val = row2.room
			} else {
				val = row2.dataMap[c.key]
			}
			if ci < len(cols)-1 {
				d.vline("BORDER", x+c.width, rBottom, y2)
			}
			if val != "" {
				layer := "DATA"
				if c.isNotes {
					layer = "NOTES"
					// Split notes into lines that fit the column width.
					dxfTextLines(d, layer, x+dxfPad, rBottom, y2, dxfTextData, c.width-2*dxfPad, val)
				} else {
					d.text(layer, x+dxfPad, rBottom+dxfRowH*dxfTextBaseFr, dxfTextData, dxfFit(val, c.width))
				}
			}
			x += c.width
		}
	}

	// Return the Y of the table's bottom edge.
	return hdrBottom - float64(len(rows))*dxfRowH
}

// dxfTextLines renders multi-line text (split on whitespace to fit colWidth)
// as multiple TEXT entities, stacked from top of cell downward.
func dxfTextLines(d *dxfWriter, layer string, x, rowBottom, rowTop, height, maxW float64, text string) {
	charsPerLine := int(maxW / dxfCharW)
	if charsPerLine < 5 {
		charsPerLine = 5
	}
	lines := dxfWrap(text, charsPerLine)
	rowH := rowTop - rowBottom
	lineSpacing := height * 1.5
	// Start near the top of the cell.
	startY := rowTop - height - (rowH-float64(len(lines))*lineSpacing)/2
	for i, line := range lines {
		y := startY - float64(i)*lineSpacing
		if y < rowBottom+height*0.2 {
			break // don't overflow the cell
		}
		d.text(layer, x, y, height, line)
	}
}

// dxfWrap wraps text to at most maxChars per line on word boundaries.
func dxfWrap(text string, maxChars int) []string {
	// Honour explicit newlines first.
	var lines []string
	for _, para := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		words := strings.Fields(para)
		if len(words) == 0 {
			continue
		}
		cur := words[0]
		for _, w := range words[1:] {
			if len(cur)+1+len(w) <= maxChars {
				cur += " " + w
			} else {
				lines = append(lines, cur)
				cur = w
			}
		}
		lines = append(lines, cur)
	}
	return lines
}

// ── column width helpers ─────────────────────────────────────────────────────

func dxfColWidthFromValues(label string, getValue func(int) string, n int, isNotes bool) float64 {
	maxLen := len(label)
	for i := 0; i < n; i++ {
		if l := len(getValue(i)); l > maxLen {
			maxLen = l
		}
	}
	maxW := dxfMaxColW
	if isNotes {
		maxW = dxfNotesMaxW
	}
	return math.Max(dxfMinColW, math.Min(maxW, float64(maxLen)*dxfCharW+2*dxfPad))
}

// dxfFit truncates s so it fits in a single-line cell of the given width.
func dxfFit(s string, colW float64) string {
	maxChars := int((colW - 2*dxfPad) / dxfCharW)
	if maxChars < 3 {
		maxChars = 3
	}
	r := []rune(s)
	if len(r) > maxChars {
		return string(r[:maxChars-1]) + "~"
	}
	return s
}

// ── low-level DXF writer ─────────────────────────────────────────────────────

type dxfWriter struct{ w io.Writer }

// pair writes one group-code / value pair.
func (d *dxfWriter) pair(code int, value any) {
	fmt.Fprintf(d.w, "%3d\n%v\n", code, value)
}

func (d *dxfWriter) header(limW, limH float64) {
	p := d.pair
	p(0, "SECTION")
	p(2, "HEADER")
	p(9, "$ACADVER"); p(1, "AC1009")
	p(9, "$INSBASE"); p(10, "0.0"); p(20, "0.0"); p(30, "0.0")
	p(9, "$LIMMIN"); p(10, "0.0"); p(20, "0.0")
	p(9, "$LIMMAX"); p(10, fmt.Sprintf("%.4f", limW)); p(20, fmt.Sprintf("%.4f", limH))
	p(9, "$LUNITS"); p(70, 2)
	p(0, "ENDSEC")
}

func (d *dxfWriter) tables() {
	p := d.pair
	p(0, "SECTION")
	p(2, "TABLES")

	// LTYPE table — must define CONTINUOUS so layers can reference it.
	p(0, "TABLE"); p(2, "LTYPE"); p(70, 1)
	p(0, "LTYPE"); p(2, "CONTINUOUS"); p(70, 0)
	p(3, "Solid line"); p(72, 65); p(73, 0); p(40, "0.0")
	p(0, "ENDTAB")

	// LAYER table.
	type lyr struct{ name, color string }
	layers := []lyr{
		{"BORDER", "7"},
		{"TITLE", "4"},
		{"COLHEAD", "2"},
		{"DATA", "7"},
		{"NOTES", "3"},
	}
	p(0, "TABLE"); p(2, "LAYER"); p(70, len(layers))
	for _, l := range layers {
		p(0, "LAYER"); p(2, l.name); p(70, 0); p(62, l.color); p(6, "CONTINUOUS")
	}
	p(0, "ENDTAB")

	p(0, "ENDSEC")
}

// blocks writes the mandatory BLOCKS section with a minimal $MODEL_SPACE entry.
func (d *dxfWriter) blocks() {
	p := d.pair
	p(0, "SECTION"); p(2, "BLOCKS")
	p(0, "BLOCK"); p(8, "0"); p(2, "$MODEL_SPACE"); p(70, 0)
	p(10, "0.0"); p(20, "0.0"); p(30, "0.0")
	p(3, "$MODEL_SPACE"); p(1, "")
	p(0, "ENDBLK"); p(8, "0")
	p(0, "ENDSEC")
}

func (d *dxfWriter) sectionStart(name string) { d.pair(0, "SECTION"); d.pair(2, name) }
func (d *dxfWriter) sectionEnd()              { d.pair(0, "ENDSEC") }
func (d *dxfWriter) eof()                     { d.pair(0, "EOF") }

func (d *dxfWriter) hline(layer string, x1, x2, y float64) {
	d.line(layer, x1, y, x2, y)
}
func (d *dxfWriter) vline(layer string, x, y1, y2 float64) {
	d.line(layer, x, y1, x, y2)
}
func (d *dxfWriter) rect(layer string, x1, y1, x2, y2 float64) {
	d.hline(layer, x1, x2, y2)
	d.hline(layer, x1, x2, y1)
	d.vline(layer, x1, y1, y2)
	d.vline(layer, x2, y1, y2)
}

func (d *dxfWriter) line(layer string, x1, y1, x2, y2 float64) {
	p := d.pair
	p(0, "LINE"); p(8, layer)
	p(10, fmt.Sprintf("%.4f", x1)); p(20, fmt.Sprintf("%.4f", y1)); p(30, "0.0")
	p(11, fmt.Sprintf("%.4f", x2)); p(21, fmt.Sprintf("%.4f", y2)); p(31, "0.0")
}

func (d *dxfWriter) text(layer string, x, y, height float64, content string) {
	if content == "" {
		return
	}
	p := d.pair
	p(0, "TEXT"); p(8, layer)
	p(10, fmt.Sprintf("%.4f", x)); p(20, fmt.Sprintf("%.4f", y)); p(30, "0.0")
	p(40, fmt.Sprintf("%.4f", height))
	p(1, dxfEscape(content))
	p(7, "STANDARD")
}

// dxfEscape strips control characters and escapes DXF special chars in TEXT strings.
func dxfEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 0x20 {
			continue
		}
		switch r {
		case '\\':
			b.WriteRune('\\'); b.WriteRune('\\')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
