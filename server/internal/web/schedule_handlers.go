package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
)

func (a app) createSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := a.queries.GetProject(r.Context(), projectID); errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.Form.Get("name"))
	if name == "" {
		http.Error(w, "schedule name is required", http.StatusBadRequest)
		return
	}

	kind := strings.TrimSpace(r.Form.Get("kind"))
	if kind == "" {
		kind = "items"
	}

	existingSchedules, err := a.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	schedule, err := a.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: projectID,
		Name:      name,
		Kind:      kind,
		Notes:     strings.TrimSpace(r.Form.Get("notes")),
		Position:  int32(len(existingSchedules) + 1),
	})
	if err != nil {
		http.Error(w, "could not create schedule", http.StatusInternalServerError)
		return
	}

	if kind == "items" {
		preset := strings.TrimSpace(r.Form.Get("preset"))
		if err := seedColumns(r.Context(), a.queries, schedule.ID, preset); err != nil {
			http.Error(w, "could not seed columns", http.StatusInternalServerError)
			return
		}
	}

	http.Redirect(w, r, "/projects/"+projectID+"#schedule-"+schedule.ID, http.StatusSeeOther)
}

func (a app) showSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	project, err := a.queries.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return
	}

	schedule, err := a.queries.GetSchedule(r.Context(), r.PathValue("scheduleID"))
	if errors.Is(err, sql.ErrNoRows) || schedule.ProjectID != project.ID {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load schedule", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+projectID+"#schedule-"+schedule.ID, http.StatusSeeOther)
}

func (a app) editSchedule(w http.ResponseWriter, r *http.Request) {
	loaded, ok := a.loadProjectSchedule(w, r, r.PathValue("projectID"), r.PathValue("scheduleID"))
	if !ok {
		return
	}

	render(w, scheduleEditPage{
		Kind:     "edit-schedule",
		Title:    "Edit " + loaded.schedule.Name,
		Project:  loaded.project,
		Schedule: loaded.schedule,
	})
}

func (a app) updateSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	loaded, ok := a.loadProjectSchedule(w, r, projectID, scheduleID)
	if !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.Form.Get("name"))
	if name == "" {
		http.Error(w, "schedule name is required", http.StatusBadRequest)
		return
	}

	kind := strings.TrimSpace(r.Form.Get("kind"))
	if kind == "" {
		kind = loaded.schedule.Kind
	}

	_, err := a.queries.UpdateSchedule(r.Context(), queries.UpdateScheduleParams{
		ID:       scheduleID,
		Name:     name,
		Kind:     kind,
		Notes:    strings.TrimSpace(r.Form.Get("notes")),
		Position: parseInt32(r.Form.Get("position"), 1),
	})
	if err != nil {
		http.Error(w, "could not update schedule", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+projectID+"#schedule-"+scheduleID, http.StatusSeeOther)
}

func (a app) deleteSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	scheduleID := r.PathValue("scheduleID")
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
		return
	}
	if err := a.queries.DeleteSchedule(r.Context(), scheduleID); err != nil {
		http.Error(w, "could not delete schedule", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects/"+projectID, http.StatusSeeOther)
}
