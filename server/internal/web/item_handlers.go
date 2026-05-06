package web

import (
	"encoding/json"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
)

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

	_, err = a.queries.CreateScheduleItem(r.Context(), queries.CreateScheduleItemParams{
		ScheduleID:     scheduleID,
		Data:           dataJSON,
		Zone:           strings.TrimSpace(r.Form.Get("col_zone")),
		SourceUrl:      strings.TrimSpace(r.Form.Get("source_url")),
		SourceTitle:    strings.TrimSpace(r.Form.Get("source_title")),
		SourceImageUrl: strings.TrimSpace(r.Form.Get("source_image_url")),
		SourcePdfLinks: splitLines(r.Form.Get("source_pdf_links")),
		Position:       int32(len(existingItems) + 1),
	})
	if err != nil {
		http.Error(w, "could not create item", http.StatusInternalServerError)
		return
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
	if err := r.ParseForm(); err != nil {
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

	_, err = a.queries.UpdateScheduleItem(r.Context(), queries.UpdateScheduleItemParams{
		ID:             item.ID,
		Data:           dataJSON,
		Zone:           strings.TrimSpace(r.Form.Get("col_zone")),
		SourceUrl:      strings.TrimSpace(r.Form.Get("source_url")),
		SourceTitle:    strings.TrimSpace(r.Form.Get("source_title")),
		SourceImageUrl: strings.TrimSpace(r.Form.Get("source_image_url")),
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
