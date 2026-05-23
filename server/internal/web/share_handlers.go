package web

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/share"
)

// newShareSlugAvoidingConflicts generates a fresh three-word slug,
// retrying up to a few times if the slug collides with an already-active
// share link in the DB. With 256³ ≈ 16M combos and typical share-link
// counts, retries should be effectively never; the safety net catches
// the long tail.
func (a app) newShareSlugAvoidingConflicts(ctx context.Context) (string, error) {
	return share.TryNewShareSlug(5, func(slug string) (bool, error) {
		if _, err := a.queries.GetActiveProjectShareLinkByHash(ctx, share.HashToken(slug)); err == nil {
			return true, nil // an active link already uses this slug
		} else if errors.Is(err, sql.ErrNoRows) {
			return false, nil // slug is free
		} else {
			return false, err // real DB error — surface it
		}
	})
}

// loadShareLinkProject resolves a share token to the underlying project and
// records the view. Returns (project, ok). On ok=false the response has
// already been written.
func (a app) loadShareLinkProject(w http.ResponseWriter, r *http.Request) (queries.Project, string, bool) {
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		renderNotFound(w)
		return queries.Project{}, "", false
	}
	link, err := a.queries.GetActiveProjectShareLinkByHash(r.Context(), share.HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		renderNotFound(w)
		return queries.Project{}, "", false
	}
	if err != nil {
		http.Error(w, "could not load share link", http.StatusInternalServerError)
		return queries.Project{}, "", false
	}
	_ = a.queries.MarkProjectShareLinkViewed(r.Context(), link.ID)

	project, err := a.queries.GetProject(r.Context(), link.ProjectID)
	if errors.Is(err, sql.ErrNoRows) {
		renderNotFound(w)
		return queries.Project{}, "", false
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return queries.Project{}, "", false
	}
	return project, token, true
}

// showPublicShareProject renders the contractor-mode project page —
// schedule list with per-schedule subtotals.
func (a app) showPublicShareProject(w http.ResponseWriter, r *http.Request) {
	project, token, ok := a.loadShareLinkProject(w, r)
	if !ok {
		return
	}
	schedules, err := a.scheduleSummariesWithContractorTotals(r.Context(), project.ID, a.contractorStaleAmberDays, a.contractorStaleRedDays)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	render(w, projectDetailPage{
		Kind:       "project",
		Title:      project.Name,
		Project:    project,
		Schedules:  schedules,
		IsOwner:    false,
		ViewMode:   "contractor",
		ShareToken: token,
	})
}

// showPublicShareSchedule renders the contractor-mode schedule page —
// items + price/lead/stock columns + subtotal block.
func (a app) showPublicShareSchedule(w http.ResponseWriter, r *http.Request) {
	project, token, ok := a.loadShareLinkProject(w, r)
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
	sw.ContractorTotals = computeContractorTotals(sw, a.contractorStaleRedDays)

	render(w, scheduleDetailPage{
		Kind:           "schedule",
		Title:          schedule.Name + " — " + project.Name,
		Project:        project,
		Schedule:       sw,
		IsOwner:        false,
		ViewMode:       "contractor",
		ShareToken:     token,
		StaleAmberDays: a.contractorStaleAmberDays,
		StaleRedDays:   a.contractorStaleRedDays,
	})
}
