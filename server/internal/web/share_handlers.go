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
	project, err := a.queries.GetProject(r.Context(), r.PathValue("projectID"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
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
	project, err := a.queries.GetProject(r.Context(), r.PathValue("projectID"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
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
	project, err := a.queries.GetProject(r.Context(), r.PathValue("projectID"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return
	}

	if err := a.queries.DeactivateProjectShareLinks(r.Context(), project.ID); err != nil {
		http.Error(w, "could not deactivate share links", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID+"/share", http.StatusSeeOther)
}

func (a app) showPublicShare(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		http.NotFound(w, r)
		return
	}

	link, err := a.queries.GetActiveProjectShareLinkByHash(r.Context(), share.HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load share link", http.StatusInternalServerError)
		return
	}
	_ = a.queries.MarkProjectShareLinkViewed(r.Context(), link.ID)

	project, err := a.queries.GetProject(r.Context(), link.ProjectID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return
	}

	schedules, err := a.schedulesWithItems(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	render(w, publicSharePage{
		Kind:      "public-share",
		Title:     project.Name,
		Project:   project,
		Schedules: schedules,
	})
}
