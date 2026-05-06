package web

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/share"

	"golang.org/x/oauth2"
)

//go:embed all:static
var staticFiles embed.FS

type Deps struct {
	Queries       *queries.Queries
	DevUserEmail  string
	DevUserName   string
	OAuthConfig   *oauth2.Config
	SessionSecret []byte
}

type app struct {
	queries       *queries.Queries
	devUserEmail  string
	devUserName   string
	oauthConfig   *oauth2.Config
	sessionSecret []byte
}

func RegisterRoutes(mux *http.ServeMux, deps Deps) {
	a := app{
		queries:       deps.Queries,
		devUserEmail:  firstNonEmpty(deps.DevUserEmail, "dev@spexxtool.local"),
		devUserName:   firstNonEmpty(deps.DevUserName, "Development User"),
		oauthConfig:   deps.OAuthConfig,
		sessionSecret: deps.SessionSecret,
	}

	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	mux.HandleFunc("GET /login", a.loginPage)
	mux.HandleFunc("GET /auth/google", a.startGoogleOAuth)
	mux.HandleFunc("GET /auth/callback", a.oauthCallback)
	mux.HandleFunc("GET /auth/done", a.authDone)
	mux.HandleFunc("POST /logout", a.logout)
	mux.HandleFunc("GET /", a.redirectHome)
	mux.HandleFunc("GET /projects", a.listProjects)
	mux.HandleFunc("POST /projects", a.createProject)
	mux.HandleFunc("GET /projects/{projectID}", a.showProject)
	mux.HandleFunc("GET /projects/{projectID}/edit", a.editProject)
	mux.HandleFunc("POST /projects/{projectID}/edit", a.updateProject)
	mux.HandleFunc("POST /projects/{projectID}/delete", a.deleteProject)
	mux.HandleFunc("POST /projects/{projectID}/schedules", a.createSchedule)
	mux.HandleFunc("POST /projects/{projectID}/notes", a.createNote)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}", a.showSchedule)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/delete", a.deleteSchedule)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items", a.createScheduleItem)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/edit", a.editScheduleItem)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/edit", a.updateScheduleItem)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/delete", a.deleteScheduleItem)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/move", a.moveScheduleItem)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}/export.csv", a.exportScheduleCSV)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/cells/{key}/edit", a.editItemCell)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/cells/{key}", a.saveItemCell)
	mux.HandleFunc("GET /projects/{projectID}/fields/{field}/edit", a.editProjectField)
	mux.HandleFunc("POST /projects/{projectID}/fields/{field}", a.saveProjectField)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}/fields/{field}/edit", a.editScheduleField)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/fields/{field}", a.saveScheduleField)
	mux.HandleFunc("GET /projects/{projectID}/share", a.manageProjectShare)
	mux.HandleFunc("POST /projects/{projectID}/share-links", a.createProjectShareLink)
	mux.HandleFunc("POST /projects/{projectID}/share-links/deactivate", a.deactivateProjectShareLinks)
	mux.HandleFunc("GET /share/{token}", a.showPublicShare)
}

func (a app) redirectHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (a app) listProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := a.requireUser(w, r)
	if !ok {
		return
	}

	projects, err := a.queries.ListProjectsByOwner(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "could not load projects", http.StatusInternalServerError)
		return
	}

	items := make([]projectListItem, 0, len(projects))
	for _, p := range projects {
		imgs, _ := a.queries.GetProjectFirstItemImages(r.Context(), p.ID)
		padded := make([]string, 4)
		for i, u := range imgs {
			if i < 4 {
				padded[i] = u
			}
		}
		items = append(items, projectListItem{Project: p, ThumbImages: padded})
	}

	render(w, projectsPage{
		Kind:     "projects",
		Title:    "Projects",
		Projects: items,
	})
}

func (a app) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := a.requireUser(w, r)
	if !ok {
		return
	}

	existing, err := a.queries.ListProjectsByOwner(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "could not load projects", http.StatusInternalServerError)
		return
	}
	taken := make(map[string]bool, len(existing))
	for _, p := range existing {
		taken[p.Name] = true
	}
	name := "New Project"
	for i := 2; taken[name]; i++ {
		name = fmt.Sprintf("New Project %d", i)
	}

	project, err := a.queries.CreateProject(r.Context(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        name,
	})
	if err != nil {
		http.Error(w, "could not create project", http.StatusInternalServerError)
		return
	}

	// Seed a default schedule so the project detail page isn't empty.
	schedule, err := a.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "New Schedule",
		Kind:      "items",
		Position:  1,
	})
	if err == nil {
		_ = seedColumns(r.Context(), a.queries, schedule.ID, "general")
	}

	http.Redirect(w, r, "/projects/"+project.ID, http.StatusSeeOther)
}

func (a app) showProject(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProject(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	schedules, err := a.schedulesWithItems(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	var firstItemImage string
outer:
	for _, sw := range schedules {
		for _, g := range sw.Groups {
			for _, item := range g.Items {
				if item.SourceImageUrl != "" {
					firstItemImage = item.SourceImageUrl
					break outer
				}
			}
		}
	}

	activeLink, err := a.queries.GetActiveProjectShareLinkByProject(r.Context(), project.ID)
	if errors.Is(err, sql.ErrNoRows) {
		if token, tokenErr := share.NewToken(); tokenErr == nil {
			if newLink, createErr := a.queries.CreateProjectShareLink(r.Context(), queries.CreateProjectShareLinkParams{
				ProjectID: project.ID,
				TokenHash: share.HashToken(token),
				Token:     token,
				Label:     "Project share",
			}); createErr == nil {
				activeLink = newLink
				err = nil
			}
		}
	}
	var activeLinkPtr *queries.ProjectShareLink
	if err == nil {
		activeLinkPtr = &activeLink
	}

	render(w, projectDetailPage{
		Kind:           "project",
		Title:          project.Name,
		Project:        project,
		Schedules:      schedules,
		FirstItemImage: firstItemImage,
		ActiveLink:     activeLinkPtr,
		ShareBaseURL:   requestBaseURL(r),
	})
}

func (a app) editProject(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProject(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	render(w, projectEditPage{
		Kind:    "edit-project",
		Title:   "Edit " + project.Name,
		Project: project,
	})
}

func (a app) updateProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, _, ok := a.loadUserProject(w, r, projectID); !ok {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(r.Form.Get("name"))
	if name == "" {
		http.Error(w, "project name is required", http.StatusBadRequest)
		return
	}

	_, err := a.queries.UpdateProject(r.Context(), queries.UpdateProjectParams{
		ID:           projectID,
		Name:         name,
		Address:      strings.TrimSpace(r.Form.Get("address")),
		Description:  strings.TrimSpace(r.Form.Get("description")),
		ThumbnailUrl: strings.TrimSpace(r.Form.Get("thumbnail_url")),
	})
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not update project", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+projectID, http.StatusSeeOther)
}

func (a app) deleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, _, ok := a.loadUserProject(w, r, projectID); !ok {
		return
	}
	if err := a.queries.DeleteProject(r.Context(), projectID); err != nil {
		http.Error(w, "could not delete project", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}
