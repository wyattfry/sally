package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/share"
)

func (a app) manageProjectShare(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	activeLink, err := a.queries.GetActiveProjectShareLinkByProject(r.Context(), project.ID)
	var activeLinkPtr *queries.ProjectShareLink
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "could not load share link", http.StatusInternalServerError)
		return
	}
	if err == nil {
		activeLinkPtr = &activeLink
	}

	render(w, shareManagePage{
		Kind:         "share-manage",
		Title:        "Share " + project.Name,
		Project:      project,
		ActiveLink:   activeLinkPtr,
		ShareBaseURL: requestBaseURL(r),
	})
}

func (a app) createProjectShareLink(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	if err := a.queries.DeactivateProjectShareLinks(r.Context(), project.ID); err != nil {
		http.Error(w, "could not deactivate existing share links", http.StatusInternalServerError)
		return
	}

	token, err := share.NewToken()
	if err != nil {
		http.Error(w, "could not create share token", http.StatusInternalServerError)
		return
	}
	_, err = a.queries.CreateProjectShareLink(r.Context(), queries.CreateProjectShareLinkParams{
		ProjectID: project.ID,
		TokenHash: share.HashToken(token),
		Token:     token,
		Label:     "Project share",
	})
	if err != nil {
		http.Error(w, "could not create share link", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/share", http.StatusSeeOther)
}

func (a app) deactivateProjectShareLinks(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	if err := a.queries.DeactivateProjectShareLinks(r.Context(), project.ID); err != nil {
		http.Error(w, "could not deactivate share links", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/share", http.StatusSeeOther)
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
