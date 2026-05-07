package web

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/schedcodes"
)

func (a app) createBlankScheduleItem(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	loaded, ok := a.loadProjectSchedule(w, r, projectID, scheduleID)
	if !ok {
		return
	}

	existingItems, err := a.queries.ListScheduleItems(r.Context(), scheduleID)
	if err != nil {
		http.Error(w, "could not load items", http.StatusInternalServerError)
		return
	}

	code := schedcodes.NextCode(existingItems, loaded.schedule.Name)
	dataJSON, _ := json.Marshal(map[string]string{"code": code})

	created, err := a.queries.CreateScheduleItem(r.Context(), queries.CreateScheduleItemParams{
		ScheduleID:     scheduleID,
		Data:           dataJSON,
		Zone:           "",
		SourceUrl:      "",
		SourceTitle:    "",
		SourceImageUrl: "",
		SourcePdfLinks: []string{},
		Position:       int32(len(existingItems) + 1),
	})
	if err != nil {
		http.Error(w, "could not create item", http.StatusInternalServerError)
		return
	}

	if r.Header.Get("HX-Request") == "true" {
		columns, _ := a.queries.ListScheduleColumns(r.Context(), scheduleID)
		w.Header().Set("Content-Type", "text/html")
		writeItemRow(w, projectID, scheduleID, created, columns)
		return
	}

	http.Redirect(w, r, "/projects/"+projectID+"#schedule-"+scheduleID, http.StatusSeeOther)
}

func (a app) createScheduleItem(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	columns, err := a.queries.ListScheduleColumns(r.Context(), scheduleID)
	if err != nil {
		http.Error(w, "could not load columns", http.StatusInternalServerError)
		return
	}

	dataMap := buildDataMap(r, columns)
	dataJSON, err := json.Marshal(dataMap)
	if err != nil {
		http.Error(w, "could not encode item data", http.StatusInternalServerError)
		return
	}

	existingItems, err := a.queries.ListScheduleItems(r.Context(), scheduleID)
	if err != nil {
		http.Error(w, "could not load items", http.StatusInternalServerError)
		return
	}

	sourceImageUrl := strings.TrimSpace(r.Form.Get("source_image_url"))
	created, err := a.queries.CreateScheduleItem(r.Context(), queries.CreateScheduleItemParams{
		ScheduleID:     scheduleID,
		Data:           dataJSON,
		Zone:           strings.TrimSpace(r.Form.Get("col_zone")),
		SourceUrl:      strings.TrimSpace(r.Form.Get("source_url")),
		SourceTitle:    strings.TrimSpace(r.Form.Get("source_title")),
		SourceImageUrl: sourceImageUrl,
		SourcePdfLinks: splitLines(r.Form.Get("source_pdf_links")),
		Position:       int32(len(existingItems) + 1),
	})
	if err != nil {
		http.Error(w, "could not create item", http.StatusInternalServerError)
		return
	}

	// Download and cache scraped images locally so they don't rely on external URLs.
	if strings.HasPrefix(sourceImageUrl, "http") && a.uploadsDir != "" {
		snap := created
		go func() {
			localURL := downloadAndSave(snap.SourceImageUrl, a.uploadsDir)
			if localURL == snap.SourceImageUrl {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = a.queries.UpdateScheduleItem(ctx, queries.UpdateScheduleItemParams{
				ID:             snap.ID,
				Data:           snap.Data,
				Zone:           snap.Zone,
				SourceUrl:      snap.SourceUrl,
				SourceTitle:    snap.SourceTitle,
				SourceImageUrl: localURL,
				SourcePdfLinks: snap.SourcePdfLinks,
				Position:       snap.Position,
			})
		}()
	}

	http.Redirect(w, r, "/projects/"+projectID+"#schedule-"+scheduleID, http.StatusSeeOther)
}

func (a app) editScheduleItem(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}

	columns, err := a.queries.ListScheduleColumns(r.Context(), loaded.schedule.ID)
	if err != nil {
		http.Error(w, "could not load columns", http.StatusInternalServerError)
		return
	}

	render(w, itemEditPage{
		Kind:     "edit-item",
		Title:    "Edit " + itemDisplayTitle(item),
		Project:  loaded.project,
		Schedule: loaded.schedule,
		Item:     toItemView(item),
		Columns:  columns,
	})
}

func (a app) updateScheduleItem(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	columns, err := a.queries.ListScheduleColumns(r.Context(), loaded.schedule.ID)
	if err != nil {
		http.Error(w, "could not load columns", http.StatusInternalServerError)
		return
	}

	dataMap := buildDataMap(r, columns)
	dataJSON, err := json.Marshal(dataMap)
	if err != nil {
		http.Error(w, "could not encode item data", http.StatusInternalServerError)
		return
	}

	sourceImageURL := strings.TrimSpace(r.Form.Get("source_image_url"))
	if uploaded, err := saveUploadedFile(r, "source_image_file", a.uploadsDir); err != nil {
		http.Error(w, "could not save image: "+err.Error(), http.StatusInternalServerError)
		return
	} else if uploaded != "" {
		sourceImageURL = uploaded
	}

	_, err = a.queries.UpdateScheduleItem(r.Context(), queries.UpdateScheduleItemParams{
		ID:             item.ID,
		Data:           dataJSON,
		Zone:           strings.TrimSpace(r.Form.Get("col_zone")),
		SourceUrl:      strings.TrimSpace(r.Form.Get("source_url")),
		SourceTitle:    strings.TrimSpace(r.Form.Get("source_title")),
		SourceImageUrl: sourceImageURL,
		SourcePdfLinks: splitLines(r.Form.Get("source_pdf_links")),
		Position:       parseInt32(r.Form.Get("position"), item.Position),
	})
	if err != nil {
		http.Error(w, "could not update item", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+loaded.project.ID+"#schedule-"+loaded.schedule.ID, http.StatusSeeOther)
}

func (a app) deleteScheduleItem(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	if err := a.queries.DeleteScheduleItem(r.Context(), item.ID); err != nil {
		http.Error(w, "could not delete item", http.StatusInternalServerError)
		return
	}
	if r.Header.Get("HX-Request") == "true" {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/projects/"+loaded.project.ID+"#schedule-"+loaded.schedule.ID, http.StatusSeeOther)
}

func (a app) moveScheduleItem(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	direction := r.Form.Get("direction")
	if direction != "up" && direction != "down" {
		http.Error(w, "invalid direction", http.StatusBadRequest)
		return
	}

	all, err := a.queries.ListScheduleItems(r.Context(), loaded.schedule.ID)
	if err != nil {
		http.Error(w, "could not load items", http.StatusInternalServerError)
		return
	}

	// Collect items in the same zone, sorted by (position, created_at) — same order as ListScheduleItems.
	var peers []queries.ScheduleItem
	for _, it := range all {
		if it.Zone == item.Zone {
			peers = append(peers, it)
		}
	}
	sort.Slice(peers, func(i, j int) bool {
		if peers[i].Position != peers[j].Position {
			return peers[i].Position < peers[j].Position
		}
		return peers[i].CreatedAt.Before(peers[j].CreatedAt)
	})

	idx := -1
	for i, p := range peers {
		if p.ID == item.ID {
			idx = i
			break
		}
	}

	var swapWith *queries.ScheduleItem
	if direction == "up" && idx > 0 {
		swapWith = &peers[idx-1]
	} else if direction == "down" && idx >= 0 && idx < len(peers)-1 {
		swapWith = &peers[idx+1]
	}

	if swapWith != nil {
		posA, posB := item.Position, swapWith.Position
		if posA == posB {
			posA, posB = int32(idx+1), int32(idx)
		}
		_ = a.queries.UpdateScheduleItemPosition(r.Context(), item.ID, posB)
		_ = a.queries.UpdateScheduleItemPosition(r.Context(), swapWith.ID, posA)
	}

	http.Redirect(w, r, "/projects/"+loaded.project.ID+"#schedule-"+loaded.schedule.ID, http.StatusSeeOther)
}

var knownDataFields = []struct{ key, label string }{
	{"code", "Code"},
	{"title", "Title"},
	{"manufacturer", "Manufacturer"},
	{"modelNumber", "Model Number"},
	{"category", "Category"},
	{"description", "Description"},
	{"finish", "Finish"},
	{"finishModelNumber", "Finish Model Number"},
	{"availableFinishes", "Available Finishes"},
	{"requiredAddOns", "Required Add-Ons"},
	{"optionalCompanions", "Optional Companions"},
	{"zone", "Zone"},
}

func (a app) showItemDetail(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	columns, _ := a.queries.ListScheduleColumns(r.Context(), loaded.schedule.ID)

	var dm map[string]any
	_ = json.Unmarshal(item.Data, &dm)
	if dm == nil {
		dm = map[string]any{}
	}
	if item.Zone != "" {
		dm["zone"] = item.Zone
	}

	// Build set of keys covered by knownDataFields for custom-column detection.
	knownKeys := make(map[string]bool, len(knownDataFields))
	for _, f := range knownDataFields {
		knownKeys[f.key] = true
	}

	var b strings.Builder
	b.WriteString(`<div class="item-detail-body">`)

	// Thumbnail
	if item.SourceImageUrl != "" {
		fmt.Fprintf(&b, `<img class="item-detail-thumb" src="%s" alt="">`, html.EscapeString(item.SourceImageUrl))
	}

	b.WriteString(`<dl class="item-detail-dl">`)

	writeField := func(label, raw string) {
		if raw == "" {
			return
		}
		esc := html.EscapeString(raw)
		var val string
		if u, err := url.ParseRequestURI(raw); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			val = fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener">%s</a>`, esc, esc)
		} else {
			val = esc
		}
		fmt.Fprintf(&b, `<dt>%s</dt><dd>%s</dd>`, html.EscapeString(label), val)
	}

	writeListField := func(label string, items []string) {
		var parts []string
		for _, s := range items {
			if s != "" {
				parts = append(parts, html.EscapeString(s))
			}
		}
		if len(parts) == 0 {
			return
		}
		fmt.Fprintf(&b, `<dt>%s</dt><dd>%s</dd>`, html.EscapeString(label), strings.Join(parts, ", "))
	}

	// Known fields in order.
	for _, f := range knownDataFields {
		v := dm[f.key]
		switch tv := v.(type) {
		case string:
			writeField(f.label, tv)
		case []any:
			strs := make([]string, 0, len(tv))
			for _, item := range tv {
				if s, ok := item.(string); ok {
					strs = append(strs, s)
				}
			}
			writeListField(f.label, strs)
		}
	}

	// Custom columns not in knownKeys.
	for _, col := range columns {
		if knownKeys[col.Key] {
			continue
		}
		if v, ok := dm[col.Key].(string); ok {
			writeField(col.Label, v)
		}
	}

	// Source fields.
	writeField("Product Page", item.SourceUrl)
	writeField("Source Title", item.SourceTitle)
	for i, link := range item.SourcePdfLinks {
		writeField(fmt.Sprintf("PDF %d", i+1), link)
	}

	b.WriteString(`</dl></div>`)

	// Footer with delete action.
	deleteURL := fmt.Sprintf("/projects/%s/schedules/%s/items/%s/delete",
		html.EscapeString(loaded.project.ID),
		html.EscapeString(loaded.schedule.ID),
		html.EscapeString(item.ID))
	fmt.Fprintf(&b, `<div class="item-detail-footer">
  <button class="button-danger"
    hx-post="%s"
    hx-target="#item-row-%s"
    hx-swap="outerHTML"
    hx-confirm="Delete this item? This cannot be undone."
    hx-on::after-request="this.closest('dialog').close()">Delete item</button>
</div>`, deleteURL, html.EscapeString(item.ID))

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, b.String())
}

func (a app) exportProjectCSV(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProject(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	schedules, err := a.schedulesWithItems(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	// Collect unique column keys across all schedules, preserving first-seen order.
	seenKeys := map[string]bool{}
	var colKeys, colLabels []string
	for _, sw := range schedules {
		for _, col := range sw.Columns {
			if !seenKeys[col.Key] {
				seenKeys[col.Key] = true
				colKeys = append(colKeys, col.Key)
				colLabels = append(colLabels, col.Label)
			}
		}
	}

	filename := strings.NewReplacer(" ", "_", "/", "-").Replace(project.Name) + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)

	cw := csv.NewWriter(w)
	_ = cw.Write(append([]string{"Schedule", "Zone"}, colLabels...))

	for _, sw := range schedules {
		if sw.Schedule.Kind == "note" {
			continue
		}
		for _, g := range sw.Groups {
			for _, item := range g.Items {
				row := []string{sw.Schedule.Name, item.Zone}
				for _, key := range colKeys {
					row = append(row, item.DataMap[key])
				}
				_ = cw.Write(row)
			}
		}
	}
	cw.Flush()
}
