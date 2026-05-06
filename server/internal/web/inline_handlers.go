package web

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
)

// ---------------------------------------------------------------------------
// Item cell inline editing
// ---------------------------------------------------------------------------

func (a app) editItemCell(w http.ResponseWriter, r *http.Request) {
	loaded, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	key := r.PathValue("key")
	value := itemCellValue(item, key)
	saveURL := itemCellURL(r, key)
	w.Header().Set("Content-Type", "text/html")
	if key == "zone" {
		allItems, err := a.queries.ListScheduleItems(r.Context(), loaded.schedule.ID)
		if err != nil {
			http.Error(w, "could not load items", http.StatusInternalServerError)
			return
		}
		writeCellEditZone(w, saveURL, value, loaded.schedule.ID, uniqueZones(allItems))
		return
	}
	writeCellEdit(w, saveURL, value, key == "notes")
}

func (a app) saveItemCell(w http.ResponseWriter, r *http.Request) {
	_, item, ok := a.loadProjectScheduleItem(w, r)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	key := r.PathValue("key")
	value := strings.TrimSpace(r.Form.Get("value"))

	var dataMap map[string]string
	if len(item.Data) > 0 {
		_ = json.Unmarshal(item.Data, &dataMap)
	}
	if dataMap == nil {
		dataMap = map[string]string{}
	}

	if key == "zone" {
		_, err := a.queries.UpdateScheduleItem(r.Context(), queries.UpdateScheduleItemParams{
			ID:             item.ID,
			Data:           item.Data,
			Zone:           value,
			SourceUrl:      item.SourceUrl,
			SourceTitle:    item.SourceTitle,
			SourceImageUrl: item.SourceImageUrl,
			SourcePdfLinks: item.SourcePdfLinks,
			Position:       item.Position,
		})
		if err != nil {
			http.Error(w, "could not update item", http.StatusInternalServerError)
			return
		}
	} else {
		if value == "" {
			delete(dataMap, key)
		} else {
			dataMap[key] = value
		}
		dataJSON, err := json.Marshal(dataMap)
		if err != nil {
			http.Error(w, "could not encode item", http.StatusInternalServerError)
			return
		}
		_, err = a.queries.UpdateScheduleItem(r.Context(), queries.UpdateScheduleItemParams{
			ID:             item.ID,
			Data:           dataJSON,
			Zone:           item.Zone,
			SourceUrl:      item.SourceUrl,
			SourceTitle:    item.SourceTitle,
			SourceImageUrl: item.SourceImageUrl,
			SourcePdfLinks: item.SourcePdfLinks,
			Position:       item.Position,
		})
		if err != nil {
			http.Error(w, "could not update item", http.StatusInternalServerError)
			return
		}
	}

	editURL := itemCellURL(r, key) + "/edit"
	displayValue := value
	if key != "zone" {
		displayValue = dataMap[key]
	}
	isCode := key == "code"
	sourceURL := ""
	if isCode {
		sourceURL = item.SourceUrl
	}

	w.Header().Set("Content-Type", "text/html")
	writeCellDisplay(w, editURL, displayValue, key, isCode, sourceURL)
}

func itemCellValue(item queries.ScheduleItem, key string) string {
	if key == "zone" {
		return item.Zone
	}
	var dm map[string]string
	_ = json.Unmarshal(item.Data, &dm)
	return dm[key]
}

func itemCellURL(r *http.Request, key string) string {
	return fmt.Sprintf("/projects/%s/schedules/%s/items/%s/cells/%s",
		r.PathValue("projectID"),
		r.PathValue("scheduleID"),
		r.PathValue("itemID"),
		key,
	)
}

func writeCellDisplay(w io.Writer, editURL, value, key string, isCode bool, sourceURL string) {
	var inner string
	switch {
	case isCode && sourceURL != "":
		inner = fmt.Sprintf(
			`<a class="code-link" href="%s" target="_blank" rel="noopener" onclick="event.stopPropagation()" title="Go to product page">%s</a><span class="code-link-icon" aria-hidden="true">↗</span>`,
			html.EscapeString(sourceURL), html.EscapeString(value))
	case isCode:
		inner = html.EscapeString(value)
	case key == "zone" && value == "":
		inner = `<div class="cell-clamp"><span class="cell-empty">—</span></div>`
	case value == "":
		inner = `<span class="cell-placeholder">Click to edit…</span>`
	default:
		inner = fmt.Sprintf(`<div class="cell-clamp">%s</div>`, html.EscapeString(value))
	}
	class := fmt.Sprintf("editable-cell col-%s", key)
	fmt.Fprintf(w, `<td class="%s" hx-get="%s" hx-trigger="click" hx-target="this" hx-swap="outerHTML">%s</td>`,
		class, html.EscapeString(editURL), inner)
}

func writeCellEditZone(w http.ResponseWriter, saveURL, value, scheduleID string, zones []string) {
	v := html.EscapeString(value)
	s := html.EscapeString(saveURL)
	listID := "zl-" + scheduleID
	esc := `onkeydown="if(event.key==='Escape'){this.value=this.dataset.original;htmx.trigger(this,'blur')}"`
	var opts strings.Builder
	opts.WriteString(`<option value="">No zone</option>`)
	for _, z := range zones {
		fmt.Fprintf(&opts, `<option value="%s">`, html.EscapeString(z))
	}
	fmt.Fprintf(w,
		`<td class="editing-cell col-zone">`+
			`<input class="cell-input" list="%s" name="value" value="%s" data-original="%s" autocomplete="off" autofocus `+
			`hx-post="%s" hx-trigger="blur, keyup[key=='Enter']" hx-target="closest td" hx-swap="outerHTML" hx-include="this" %s>`+
			`<datalist id="%s">%s</datalist></td>`,
		listID, v, v, s, esc, listID, opts.String())
}

func uniqueZones(items []queries.ScheduleItem) []string {
	seen := map[string]bool{}
	var out []string
	for _, it := range items {
		if it.Zone != "" && !seen[it.Zone] {
			seen[it.Zone] = true
			out = append(out, it.Zone)
		}
	}
	return out
}

func writeCellEdit(w http.ResponseWriter, saveURL, value string, multiline bool) {
	v := html.EscapeString(value)
	s := html.EscapeString(saveURL)
	esc := `onkeydown="if(event.key==='Escape'){this.value=this.dataset.original;htmx.trigger(this,'blur')}"`
	if multiline {
		fmt.Fprintf(w, `<td class="editing-cell"><textarea class="cell-input cell-textarea" name="value" data-original="%s" autofocus `+
			`hx-post="%s" hx-trigger="blur" hx-target="closest td" hx-swap="outerHTML" hx-include="this" %s>%s</textarea></td>`,
			v, s, esc, v)
	} else {
		fmt.Fprintf(w, `<td class="editing-cell"><input class="cell-input" name="value" value="%s" data-original="%s" autofocus `+
			`hx-post="%s" hx-trigger="blur, keyup[key=='Enter']" hx-target="closest td" hx-swap="outerHTML" hx-include="this" `+
			`onkeydown="if(event.key==='Escape'){this.value=this.dataset.original;htmx.trigger(this,'blur')}"></td>`,
			v, v, s)
	}
}

// ---------------------------------------------------------------------------
// Project field inline editing
// ---------------------------------------------------------------------------

func (a app) editProjectField(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}
	field := r.PathValue("field")
	saveURL := fmt.Sprintf("/projects/%s/fields/%s", project.ID, field)
	value := projectFieldValue(project, field)
	w.Header().Set("Content-Type", "text/html")
	writeProjectFieldEdit(w, field, saveURL, value)
}

func (a app) saveProjectField(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	field := r.PathValue("field")
	value := strings.TrimSpace(r.Form.Get("value"))

	params := queries.UpdateProjectParams{
		ID:           project.ID,
		Name:         project.Name,
		Address:      project.Address,
		Description:  project.Description,
		ThumbnailUrl: project.ThumbnailUrl,
	}
	switch field {
	case "name":
		if value == "" {
			value = project.Name // name is required; keep old on blank
		}
		params.Name = value
	case "address":
		params.Address = value
	case "description":
		params.Description = value
	default:
		http.Error(w, "unknown field", http.StatusBadRequest)
		return
	}

	updated, err := a.queries.UpdateProject(r.Context(), params)
	if err != nil {
		http.Error(w, "could not update project", http.StatusInternalServerError)
		return
	}

	editURL := fmt.Sprintf("/projects/%s/fields/%s/edit", project.ID, field)
	w.Header().Set("Content-Type", "text/html")
	writeProjectFieldDisplay(w, field, editURL, projectFieldValue(updated, field))
}

func projectFieldValue(p queries.Project, field string) string {
	switch field {
	case "name":
		return p.Name
	case "address":
		return p.Address
	case "description":
		return p.Description
	}
	return ""
}

func writeProjectFieldDisplay(w http.ResponseWriter, field, editURL, value string) {
	e := html.EscapeString
	switch field {
	case "name":
		fmt.Fprintf(w,
			`<h1 class="editable-h1" hx-get="%s" hx-trigger="click" hx-target="this" hx-swap="outerHTML">%s</h1>`,
			e(editURL), e(value))
	default:
		label := strings.Title(field) //nolint:staticcheck
		var valueHTML string
		if value == "" {
			valueHTML = `<span class="meta-placeholder">Click to edit…</span>`
		} else {
			valueHTML = fmt.Sprintf(`<span class="meta-value">%s</span>`, e(value))
		}
		fmt.Fprintf(w,
			`<div class="meta-field editable-meta" hx-get="%s" hx-trigger="click" hx-target="this" hx-swap="outerHTML">`+
				`<span class="meta-label">%s</span>%s</div>`,
			e(editURL), e(label), valueHTML)
	}
}

func writeProjectFieldEdit(w http.ResponseWriter, field, saveURL, value string) {
	e := html.EscapeString
	v := e(value)
	s := e(saveURL)
	esc := `onkeydown="if(event.key==='Escape'){this.value=this.dataset.original;htmx.trigger(this,'blur')}"`
	switch field {
	case "name":
		trigger := `hx-trigger="blur, keyup[key=='Enter']" hx-target="closest [data-field]" hx-swap="outerHTML" hx-include="this"`
		fmt.Fprintf(w,
			`<h1 class="editable-h1 editing-h1" data-field="name">`+
				`<input class="h1-input" name="value" value="%s" data-original="%s" autofocus `+
				`hx-post="%s" %s %s></h1>`,
			v, v, s, trigger, esc)
	case "description":
		label := strings.Title(field) //nolint:staticcheck
		fmt.Fprintf(w,
			`<div class="meta-field editing-meta" data-field="%s">`+
				`<span class="meta-label">%s</span>`+
				`<textarea class="meta-input meta-textarea" name="value" data-original="%s" autofocus `+
				`hx-post="%s" hx-trigger="blur" hx-target="closest [data-field]" hx-swap="outerHTML" hx-include="this" %s>%s</textarea></div>`,
			e(field), e(label), v, s, esc, v)
	default:
		trigger := `hx-trigger="blur, keyup[key=='Enter']" hx-target="closest [data-field]" hx-swap="outerHTML" hx-include="this"`
		label := strings.Title(field) //nolint:staticcheck
		fmt.Fprintf(w,
			`<div class="meta-field editing-meta" data-field="%s">`+
				`<span class="meta-label">%s</span>`+
				`<input class="meta-input" name="value" value="%s" data-original="%s" autofocus `+
				`hx-post="%s" %s %s></div>`,
			e(field), e(label), v, v, s, trigger, esc)
	}
}

// ---------------------------------------------------------------------------
// Schedule field inline editing (name + notes)
// ---------------------------------------------------------------------------

func (a app) editScheduleField(w http.ResponseWriter, r *http.Request) {
	loaded, ok := a.loadProjectSchedule(w, r, r.PathValue("projectID"), r.PathValue("scheduleID"))
	if !ok {
		return
	}
	field := r.PathValue("field")
	saveURL := fmt.Sprintf("/projects/%s/schedules/%s/fields/%s", loaded.project.ID, loaded.schedule.ID, field)
	w.Header().Set("Content-Type", "text/html")
	switch field {
	case "name":
		writeScheduleNameEdit(w, saveURL, loaded.schedule.Name, loaded.schedule.ID)
	case "notes":
		writeScheduleNotesEdit(w, saveURL, loaded.schedule.Notes)
	default:
		http.Error(w, "unknown field", http.StatusBadRequest)
	}
}

func (a app) saveScheduleField(w http.ResponseWriter, r *http.Request) {
	loaded, ok := a.loadProjectSchedule(w, r, r.PathValue("projectID"), r.PathValue("scheduleID"))
	if !ok {
		return
	}
	field := r.PathValue("field")
	if field != "name" && field != "notes" {
		http.Error(w, "unknown field", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	value := strings.TrimSpace(r.Form.Get("value"))

	name := loaded.schedule.Name
	notes := loaded.schedule.Notes
	switch field {
	case "name":
		if value == "" {
			value = loaded.schedule.Name
		}
		name = value
	case "notes":
		notes = value
	}

	_, err := a.queries.UpdateSchedule(r.Context(), queries.UpdateScheduleParams{
		ID:       loaded.schedule.ID,
		Name:     name,
		Kind:     loaded.schedule.Kind,
		Notes:    notes,
		Position: loaded.schedule.Position,
	})
	if err != nil {
		http.Error(w, "could not update schedule", http.StatusInternalServerError)
		return
	}

	nameEditURL := fmt.Sprintf("/projects/%s/schedules/%s/fields/name/edit", loaded.project.ID, loaded.schedule.ID)
	notesEditURL := fmt.Sprintf("/projects/%s/schedules/%s/fields/notes/edit", loaded.project.ID, loaded.schedule.ID)
	w.Header().Set("Content-Type", "text/html")
	switch field {
	case "name":
		writeScheduleNameDisplay(w, nameEditURL, name, loaded.schedule.ID)
	case "notes":
		writeScheduleNotesDisplay(w, notesEditURL, notes)
	}
}

func writeScheduleNameDisplay(w http.ResponseWriter, editURL, value, scheduleID string) {
	e := html.EscapeString
	fmt.Fprintf(w,
		`<h2 id="sname-%s" class="schedule-name" data-field="name" `+
			`hx-get="%s" hx-trigger="click" hx-target="this" hx-swap="outerHTML">%s</h2>`,
		e(scheduleID), e(editURL), e(value))
}

func writeScheduleNameEdit(w http.ResponseWriter, saveURL, value, scheduleID string) {
	e := html.EscapeString
	v := e(value)
	s := e(saveURL)
	esc := `onkeydown="if(event.key==='Escape'){this.value=this.dataset.original;htmx.trigger(this,'blur')}"`
	fmt.Fprintf(w,
		`<h2 id="sname-%s" class="schedule-name editing-schedule-name" data-field="name">`+
			`<input class="schedule-name-input" name="value" value="%s" data-original="%s" autofocus `+
			`hx-post="%s" hx-trigger="blur, keyup[key=='Enter']" hx-target="closest [data-field]" hx-swap="outerHTML" hx-include="this" %s></h2>`,
		e(scheduleID), v, v, s, esc)
}

func nl2br(s string) string {
	s = html.EscapeString(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

func writeScheduleNotesDisplay(w http.ResponseWriter, editURL, value string) {
	e := html.EscapeString
	var inner string
	if value == "" {
		inner = `<span class="notes-placeholder">Add notes…</span>`
	} else {
		inner = fmt.Sprintf(`<span class="schedule-notes-text">%s</span>`, nl2br(value))
	}
	fmt.Fprintf(w,
		`<div class="schedule-notes editable-notes" data-field="notes" `+
			`hx-get="%s" hx-trigger="click" hx-target="this" hx-swap="outerHTML">%s</div>`,
		e(editURL), inner)
}

func writeScheduleNotesEdit(w http.ResponseWriter, saveURL, value string) {
	e := html.EscapeString
	v := e(value)
	s := e(saveURL)
	rows := strings.Count(value, "\n") + 2
	if rows < 3 {
		rows = 3
	}
	esc := `onkeydown="if(event.key==='Escape'){this.value=this.dataset.original;htmx.trigger(this,'blur')}"`
	fmt.Fprintf(w,
		`<div class="schedule-notes editing-notes" data-field="notes">`+
			`<textarea class="schedule-notes-input" name="value" rows="%d" data-original="%s" autofocus `+
			`hx-post="%s" hx-trigger="blur" hx-target="closest [data-field]" hx-swap="outerHTML" hx-include="this" %s>%s</textarea></div>`,
		rows, v, s, esc, v)
}

// writeItemRow writes a complete <tr class="item-row"> fragment, used when
// adding a blank item inline via HTMX without a full page reload.
func writeItemRow(w io.Writer, projectID, scheduleID string, item queries.ScheduleItem, columns []queries.ScheduleColumn) {
	var dm map[string]string
	_ = json.Unmarshal(item.Data, &dm)
	if dm == nil {
		dm = map[string]string{}
	}
	dm["zone"] = item.Zone

	thumbUploadURL := fmt.Sprintf("/projects/%s/schedules/%s/items/%s/thumbnail", projectID, scheduleID, item.ID)
	moveURL := fmt.Sprintf("/projects/%s/schedules/%s/items/%s/move", projectID, scheduleID, item.ID)
	deleteURL := fmt.Sprintf("/projects/%s/schedules/%s/items/%s/delete", projectID, scheduleID, item.ID)
	e := html.EscapeString

	fmt.Fprintf(w, `<tr class="item-row">`)
	writeItemThumbCell(w, item.ID, item.SourceImageUrl, thumbUploadURL)

	for _, col := range columns {
		cellEditURL := fmt.Sprintf("/projects/%s/schedules/%s/items/%s/cells/%s/edit",
			projectID, scheduleID, item.ID, col.Key)
		isCode := col.Key == "code"
		sourceURL := ""
		if isCode {
			sourceURL = item.SourceUrl
		}
		writeCellDisplay(w, cellEditURL, dm[col.Key], col.Key, isCode, sourceURL)
	}

	fmt.Fprintf(w,
		`<td class="row-actions">`+
			`<form method="post" action="%s" class="move-form"><input type="hidden" name="direction" value="up"><button type="submit" class="move-btn" title="Move up">↑</button></form>`+
			`<form method="post" action="%s" class="move-form"><input type="hidden" name="direction" value="down"><button type="submit" class="move-btn" title="Move down">↓</button></form>`+
			`<button class="delete-row-btn" hx-post="%s" hx-target="closest tr" hx-swap="outerHTML" hx-confirm="Remove this item?">×</button>`+
			`</td>`,
		e(moveURL), e(moveURL), e(deleteURL))
	fmt.Fprintf(w, `</tr>`)
}
