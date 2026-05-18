package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	queries "sally/server/internal/db/generated"
)

func (a app) createSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, _, ok := a.loadUserProject(w, r, projectID); !ok {
		return
	}

	existing, err := a.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	taken := make(map[string]bool, len(existing))
	for _, s := range existing {
		taken[s.Name] = true
	}
	name := "New Schedule"
	for i := 2; taken[name]; i++ {
		name = fmt.Sprintf("New Schedule %d", i)
	}

	schedule, err := a.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: projectID,
		Name:      name,
		Kind:      "items",
		Position:  int32(len(existing) + 1),
	})
	if err != nil {
		http.Error(w, "could not create schedule", http.StatusInternalServerError)
		return
	}

	if err := seedColumns(r.Context(), a.queries, schedule.ID, "general"); err != nil {
		http.Error(w, "could not seed columns", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+projectID+"/schedules/"+schedule.ID, http.StatusSeeOther)
}

func (a app) createNote(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, _, ok := a.loadUserProject(w, r, projectID); !ok {
		return
	}

	existing, err := a.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	taken := make(map[string]bool, len(existing))
	for _, s := range existing {
		taken[s.Name] = true
	}
	name := "New Note"
	for i := 2; taken[name]; i++ {
		name = fmt.Sprintf("New Note %d", i)
	}

	schedule, err := a.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: projectID,
		Name:      name,
		Kind:      "note",
		Position:  int32(len(existing) + 1),
	})
	if err != nil {
		http.Error(w, "could not create note", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+projectID+"/schedules/"+schedule.ID, http.StatusSeeOther)
}

func (a app) showSchedule(w http.ResponseWriter, r *http.Request) {
	user, project, ok := a.loadUserProject(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}
	schedule, err := a.queries.GetSchedule(r.Context(), r.PathValue("scheduleID"))
	if errors.Is(err, sql.ErrNoRows) || schedule.ProjectID != project.ID {
		renderNotFound(w)
		return
	}
	if err != nil {
		http.Error(w, "could not load schedule", http.StatusInternalServerError)
		return
	}

	sw, err := a.scheduleWithItemsByID(r.Context(), schedule)
	if err != nil {
		http.Error(w, "could not load schedule", http.StatusInternalServerError)
		return
	}

	isOwner := project.OwnerUserID == user.ID || a.oauthConfig == nil
	var ownerDisplayName string
	if !isOwner {
		if owner, ownerErr := a.queries.GetUser(r.Context(), project.OwnerUserID); ownerErr == nil {
			ownerDisplayName = owner.Name
			if ownerDisplayName == "" {
				ownerDisplayName = owner.Email
			}
		}
	}

	render(w, scheduleDetailPage{
		Kind:             "schedule",
		Title:            schedule.Name + " — " + project.Name,
		Project:          project,
		Schedule:         sw,
		IsOwner:          isOwner,
		OwnerDisplayName: ownerDisplayName,
	})
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
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/projects/"+projectID)
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/projects/"+projectID, http.StatusSeeOther)
}

func (a app) reorderSchedules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, _, ok := a.loadUserProject(w, r, projectID); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	for i, id := range r.Form["ids"] {
		_ = a.queries.UpdateSchedulePosition(r.Context(), queries.UpdateSchedulePositionParams{ID: id, Position: int32(i + 1)})
	}
	w.WriteHeader(http.StatusNoContent)
}
