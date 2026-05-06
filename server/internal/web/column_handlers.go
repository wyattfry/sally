package web

import (
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"

	queries "sally/server/internal/db/generated"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func labelToKey(label string) string {
	s := strings.ToLower(strings.TrimSpace(label))
	s = nonAlnum.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "col"
	}
	return s
}

func (a app) addScheduleColumn(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	label := strings.TrimSpace(r.Form.Get("label"))
	if label == "" {
		http.Error(w, "label is required", http.StatusBadRequest)
		return
	}

	existing, err := a.queries.ListScheduleColumns(r.Context(), scheduleID)
	if err != nil {
		http.Error(w, "could not load columns", http.StatusInternalServerError)
		return
	}

	// Ensure key uniqueness within this schedule.
	baseKey := labelToKey(label)
	key := baseKey
	taken := map[string]bool{}
	for _, c := range existing {
		taken[c.Key] = true
	}
	for i := 2; taken[key]; i++ {
		key = fmt.Sprintf("%s_%d", baseKey, i)
	}

	col, err := a.queries.CreateScheduleColumn(r.Context(), queries.CreateScheduleColumnParams{
		ScheduleID: scheduleID,
		Key:        key,
		Label:      label,
		Kind:       "text",
		Position:   int32(len(existing) + 1),
	})
	if err != nil {
		http.Error(w, "could not create column", http.StatusInternalServerError)
		return
	}

	// Return an HTML fragment for HTMX to append into the column list.
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<li class="col-modal-row" id="col-row-%s" data-col-id="%s">
  <div class="col-modal-move-btns">
    <button class="col-modal-move" type="button" onclick="moveColRow(this,-1)" title="Move up">↑</button>
    <button class="col-modal-move" type="button" onclick="moveColRow(this,1)" title="Move down">↓</button>
  </div>
  <input class="col-modal-label" type="text" value="%s"
    hx-post="/projects/%s/schedules/%s/columns/%s/rename"
    hx-trigger="change" hx-swap="none"
    hx-on::after-request="window.__columnsChanged=true"
    name="label">
  <button class="col-modal-delete" type="button"
    hx-post="/projects/%s/schedules/%s/columns/%s/delete"
    hx-target="#col-row-%s" hx-swap="delete"
    hx-confirm="Delete column &#39;%s&#39;? Existing data in this column will be hidden but not lost."
    hx-on::before-request="window.__columnsChanged=true">Delete</button>
</li>`,
		template.HTMLEscapeString(col.ID),
		template.HTMLEscapeString(col.ID),
		template.HTMLEscapeString(col.Label),
		template.HTMLEscapeString(projectID),
		template.HTMLEscapeString(scheduleID),
		template.HTMLEscapeString(col.ID),
		template.HTMLEscapeString(projectID),
		template.HTMLEscapeString(scheduleID),
		template.HTMLEscapeString(col.ID),
		template.HTMLEscapeString(col.ID),
		template.HTMLEscapeString(col.Label),
	)
}

func (a app) reorderScheduleColumns(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	ids := r.Form["ids"]
	for i, id := range ids {
		_ = a.queries.UpdateScheduleColumnPosition(r.Context(), id, int32(i+1))
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a app) renameScheduleColumn(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	columnID := r.PathValue("columnID")
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	label := strings.TrimSpace(r.Form.Get("label"))
	if label == "" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := a.queries.UpdateScheduleColumnLabel(r.Context(), columnID, label); err != nil {
		http.Error(w, "could not rename column", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a app) deleteScheduleColumn(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	columnID := r.PathValue("columnID")
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
		return
	}

	if err := a.queries.DeleteScheduleColumn(r.Context(), columnID); err != nil {
		http.Error(w, "could not delete column", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
