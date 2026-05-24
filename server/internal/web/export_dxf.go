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
	dxfMarginX    = 0.5  // left/right page margin
	dxfMarginY    = 0.5  // bottom margin
	dxfGapY       = 0.75 // vertical gap between consecutive schedule tables
	dxfTitleH     = 0.50 // title row height
	dxfHeaderH    = 0.36 // column-label row height
	dxfMinRowH    = 0.36 // minimum data row height
	dxfLineSpace  = 0.15 // line spacing for wrapped text
	dxfTextTitle  = 0.18 // title text height
	dxfTextHeader = 0.10 // column-label text height
	dxfTextData   = 0.10 // cell text height
	dxfPad        = 0.08 // horizontal padding inside a cell
	dxfVPad       = 0.08 // vertical padding: baseline from cell bottom
	dxfMinColW    = 1.00 // minimum column width
	dxfMaxColW    = 3.50 // normal maximum column width
	dxfNotesMaxW  = 5.00 // notes/description column maximum
	// dxfCharW: effective width per character for STANDARD (simplex) font
	// at dxfTextData height, accounting for inter-character spacing.
	// Slightly conservative so lines never overflow their column boundary.
	dxfCharW = 0.07
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

// dxfRow holds one item's flat display values.
type dxfRow struct {
	room    string
	dataMap map[string]string
}

// dxfTablePlan pre-computes columns and per-row heights so we can size the
// drawing canvas before emitting any entities.
type dxfTablePlan struct {
	sw      scheduleWithItems
	rows    []dxfRow
	cols    []dxfCol
	tableW  float64
	rowH    []float64 // per-row height (varies with notes line count)
	totalH  float64   // title + header + all data rows
}

type dxfCol struct {
	key, label string
	width      float64
	isNotes    bool
}

func dxfPlanTable(sw scheduleWithItems) dxfTablePlan {
	var rows []dxfRow
	for _, g := range sw.Groups {
		for _, it := range g.Items {
			rows = append(rows, dxfRow{room: it.Room, dataMap: it.DataMap})
		}
	}

	var cols []dxfCol
	hasRoom := false
	for _, r2 := range rows {
		if r2.room != "" {
			hasRoom = true
			break
		}
	}
	if hasRoom {
		w2 := dxfColWidthFromValues("Room", func(i int) string { return rows[i].room }, len(rows), false)
		cols = append(cols, dxfCol{key: "room", label: "Room", width: w2})
	}
	for _, sc := range sw.Columns {
		if sc.Key == "room" {
			continue
		}
		notes := sc.Key == "notes" || sc.Key == "description"
		w2 := dxfColWidthFromValues(sc.Label, func(i int) string { return rows[i].dataMap[sc.Key] }, len(rows), notes)
		cols = append(cols, dxfCol{key: sc.Key, label: sc.Label, width: w2, isNotes: notes})
	}

	tableW := 0.0
	for _, c := range cols {
		tableW += c.width
	}

	// Per-row height: expand to fit wrapped notes.
	rowH := make([]float64, len(rows))
	for ri, row2 := range rows {
		h := dxfMinRowH
		for _, c := range cols {
			if !c.isNotes {
				continue
			}
			var val string
			if c.key == "room" {
				val = row2.room
			} else {
				val = row2.dataMap[c.key]
			}
			if val == "" {
				continue
			}
			charsPerLine := int((c.width - 2*dxfPad) / dxfCharW)
			if charsPerLine < 5 {
				charsPerLine = 5
			}
			nLines := len(dxfWrap(val, charsPerLine))
			need := dxfVPad + float64(nLines)*dxfLineSpace + dxfVPad
			if need > h {
				h = need
			}
		}
		rowH[ri] = h
	}

	totalH := dxfTitleH + dxfHeaderH
	for _, h := range rowH {
		totalH += h
	}

	return dxfTablePlan{sw: sw, rows: rows, cols: cols, tableW: tableW, rowH: rowH, totalH: totalH}
}

func dxfWriteProject(w io.Writer, projectName string, tables []scheduleWithItems) {
	d := &dxfWriter{w: w}

	plans := make([]dxfTablePlan, len(tables))
	for i, sw := range tables {
		plans[i] = dxfPlanTable(sw)
	}

	totalH := dxfMarginY
	maxW := 0.0
	for i, p := range plans {
		totalH += p.totalH
		if p.tableW > maxW {
			maxW = p.tableW
		}
		if i < len(plans)-1 {
			totalH += dxfGapY
		}
	}
	totalH += dxfMarginY
	totalW := dxfMarginX + maxW + dxfMarginX

	d.header(totalW, totalH)
	d.tables()
	d.blocks()
	d.sectionStart("ENTITIES")

	curY := totalH - dxfMarginY
	for _, p := range plans {
		curY = dxfDrawTable(d, p, curY)
		curY -= dxfGapY
	}

	d.sectionEnd()
	d.eof()
}

// dxfDrawTable draws one schedule table and returns the Y of its bottom edge.
func dxfDrawTable(d *dxfWriter, p dxfTablePlan, topY float64) float64 {
	if len(p.cols) == 0 {
		return topY
	}
	x0 := dxfMarginX

	// ── Title row ─────────────────────────────────────────────────────────
	yBottom := topY - dxfTitleH
	d.rect("BORDER", x0, yBottom, x0+p.tableW, topY)
	d.text("TITLE", x0+dxfPad, yBottom+dxfVPad, dxfTextTitle, p.sw.Schedule.Name)

	// ── Column header row ─────────────────────────────────────────────────
	y2 := yBottom
	hdrBottom := y2 - dxfHeaderH
	d.hline("BORDER", x0, x0+p.tableW, hdrBottom)
	d.vline("BORDER", x0, hdrBottom, y2)
	d.vline("BORDER", x0+p.tableW, hdrBottom, y2)
	x := x0
	for i, c := range p.cols {
		d.text("COLHEAD", x+dxfPad, hdrBottom+dxfVPad, dxfTextHeader, c.label)
		if i < len(p.cols)-1 {
			d.vline("BORDER", x+c.width, hdrBottom, y2)
		}
		x += c.width
	}

	// ── Data rows ─────────────────────────────────────────────────────────
	curY := hdrBottom
	for ri, row2 := range p.rows {
		rH := p.rowH[ri]
		rBottom := curY - rH
		d.hline("BORDER", x0, x0+p.tableW, rBottom)
		d.vline("BORDER", x0, rBottom, curY)
		d.vline("BORDER", x0+p.tableW, rBottom, curY)
		x = x0
		for ci, c := range p.cols {
			var val string
			if c.key == "room" {
				val = row2.room
			} else {
				val = row2.dataMap[c.key]
			}
			if ci < len(p.cols)-1 {
				d.vline("BORDER", x+c.width, rBottom, curY)
			}
			if val != "" {
				if c.isNotes {
					dxfTextLines(d, "NOTES", x+dxfPad, rBottom, curY, dxfTextData, c.width-2*dxfPad, val)
				} else {
					d.text("DATA", x+dxfPad, rBottom+dxfVPad, dxfTextData, dxfFit(val, c.width))
				}
			}
			x += c.width
		}
		curY = rBottom
	}

	return curY
}

// dxfTextLines renders wrapped text as stacked TEXT entities, top-to-bottom.
func dxfTextLines(d *dxfWriter, layer string, x, rowBottom, rowTop, height, maxW float64, text string) {
	charsPerLine := int(maxW / dxfCharW)
	if charsPerLine < 5 {
		charsPerLine = 5
	}
	lines := dxfWrap(text, charsPerLine)
	// Place first line just below the top of the cell.
	startY := rowTop - height - dxfVPad
	for i, line := range lines {
		y := startY - float64(i)*dxfLineSpace
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
		{"TITLE", "7"},
		{"COLHEAD", "7"},
		{"DATA", "7"},
		{"NOTES", "7"},
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
