package web

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	queries "sally/server/internal/db/generated"
	"sally/server/internal/share"
)

//go:embed static/app.css
var appCSS string

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

	mux.HandleFunc("GET /static/app.css", serveAppCSS)
	mux.HandleFunc("GET /login", a.loginPage)
	mux.HandleFunc("GET /auth/google", a.startGoogleOAuth)
	mux.HandleFunc("GET /auth/callback", a.oauthCallback)
	mux.HandleFunc("POST /logout", a.logout)
	mux.HandleFunc("GET /", a.redirectHome)
	mux.HandleFunc("GET /projects", a.listProjects)
	mux.HandleFunc("GET /projects/new", a.newProject)
	mux.HandleFunc("POST /projects", a.createProject)
	mux.HandleFunc("GET /projects/{projectID}", a.showProject)
	mux.HandleFunc("GET /projects/{projectID}/edit", a.editProject)
	mux.HandleFunc("POST /projects/{projectID}/edit", a.updateProject)
	mux.HandleFunc("POST /projects/{projectID}/delete", a.deleteProject)
	mux.HandleFunc("POST /projects/{projectID}/schedules", a.createSchedule)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}", a.showSchedule)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}/edit", a.editSchedule)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/edit", a.updateSchedule)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/delete", a.deleteSchedule)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items", a.createScheduleItem)
	mux.HandleFunc("GET /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/edit", a.editScheduleItem)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/edit", a.updateScheduleItem)
	mux.HandleFunc("POST /projects/{projectID}/schedules/{scheduleID}/items/{itemID}/delete", a.deleteScheduleItem)
	mux.HandleFunc("GET /projects/{projectID}/share", a.manageProjectShare)
	mux.HandleFunc("POST /projects/{projectID}/share-links", a.createProjectShareLink)
	mux.HandleFunc("POST /projects/{projectID}/share-links/deactivate", a.deactivateProjectShareLinks)
	mux.HandleFunc("GET /share/{token}", a.showPublicShare)
}

func serveAppCSS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(appCSS))
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

	render(w, projectsPage{
		Kind:     "projects",
		Title:    "Projects",
		Projects: projects,
	})
}

func (a app) newProject(w http.ResponseWriter, _ *http.Request) {
	render(w, projectFormPage{Kind: "new-project", Title: "New Project"})
}

func (a app) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := a.requireUser(w, r)
	if !ok {
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

	project, err := a.queries.CreateProject(r.Context(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        name,
		Address:     strings.TrimSpace(r.Form.Get("address")),
	})
	if err != nil {
		http.Error(w, "could not create project", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID, http.StatusSeeOther)
}

func (a app) showProject(w http.ResponseWriter, r *http.Request) {
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

	schedules, err := a.schedulesWithItems(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	render(w, projectDetailPage{
		Kind:      "project",
		Title:     project.Name,
		Project:   project,
		Schedules: schedules,
	})
}

func (a app) editProject(w http.ResponseWriter, r *http.Request) {
	project, err := a.queries.GetProject(r.Context(), r.PathValue("projectID"))
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
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
		ID:      projectID,
		Name:    name,
		Address: strings.TrimSpace(r.Form.Get("address")),
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
	if err := a.queries.DeleteProject(r.Context(), projectID); err != nil {
		http.Error(w, "could not delete project", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

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

	existingSchedules, err := a.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	schedule, err := a.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: projectID,
		Name:      name,
		Position:  int32(len(existingSchedules) + 1),
	})
	if err != nil {
		http.Error(w, "could not create schedule", http.StatusInternalServerError)
		return
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
	if _, ok := a.loadProjectSchedule(w, r, projectID, scheduleID); !ok {
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

	_, err := a.queries.UpdateSchedule(r.Context(), queries.UpdateScheduleParams{
		ID:       scheduleID,
		Name:     name,
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

	title := strings.TrimSpace(r.Form.Get("title"))
	if title == "" {
		http.Error(w, "item title is required", http.StatusBadRequest)
		return
	}

	existingItems, err := a.queries.ListScheduleItems(r.Context(), scheduleID)
	if err != nil {
		http.Error(w, "could not load items", http.StatusInternalServerError)
		return
	}

	_, err = a.queries.CreateScheduleItem(r.Context(), queries.CreateScheduleItemParams{
		ScheduleID:        scheduleID,
		Code:              strings.TrimSpace(r.Form.Get("code")),
		Title:             title,
		Description:       strings.TrimSpace(r.Form.Get("description")),
		Manufacturer:      strings.TrimSpace(r.Form.Get("manufacturer")),
		ModelNumber:       strings.TrimSpace(r.Form.Get("model_number")),
		Finish:            strings.TrimSpace(r.Form.Get("finish")),
		FinishModelNumber: strings.TrimSpace(r.Form.Get("finish_model_number")),
		Notes:             strings.TrimSpace(r.Form.Get("notes")),
		SourceUrl:         strings.TrimSpace(r.Form.Get("source_url")),
		SourceTitle:       strings.TrimSpace(r.Form.Get("source_title")),
		SourceImageUrl:    strings.TrimSpace(r.Form.Get("source_image_url")),
		SourcePdfLinks:    splitLines(r.Form.Get("source_pdf_links")),
		Position:          int32(len(existingItems) + 1),
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

	render(w, itemEditPage{
		Kind:     "edit-item",
		Title:    "Edit " + item.Title,
		Project:  loaded.project,
		Schedule: loaded.schedule,
		Item:     item,
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

	title := strings.TrimSpace(r.Form.Get("title"))
	if title == "" {
		http.Error(w, "item title is required", http.StatusBadRequest)
		return
	}

	_, err := a.queries.UpdateScheduleItem(r.Context(), queries.UpdateScheduleItemParams{
		ID:                item.ID,
		Code:              strings.TrimSpace(r.Form.Get("code")),
		Title:             title,
		Description:       strings.TrimSpace(r.Form.Get("description")),
		Manufacturer:      strings.TrimSpace(r.Form.Get("manufacturer")),
		ModelNumber:       strings.TrimSpace(r.Form.Get("model_number")),
		Finish:            strings.TrimSpace(r.Form.Get("finish")),
		FinishModelNumber: strings.TrimSpace(r.Form.Get("finish_model_number")),
		Notes:             strings.TrimSpace(r.Form.Get("notes")),
		SourceUrl:         strings.TrimSpace(r.Form.Get("source_url")),
		SourceTitle:       strings.TrimSpace(r.Form.Get("source_title")),
		SourceImageUrl:    strings.TrimSpace(r.Form.Get("source_image_url")),
		SourcePdfLinks:    splitLines(r.Form.Get("source_pdf_links")),
		Position:          parseInt32(r.Form.Get("position"), item.Position),
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
	http.Redirect(w, r, "/projects/"+loaded.project.ID+"#schedule-"+loaded.schedule.ID, http.StatusSeeOther)
}

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

type projectSchedule struct {
	project  queries.Project
	schedule queries.Schedule
}

func (a app) loadProjectSchedule(w http.ResponseWriter, r *http.Request, projectID string, scheduleID string) (projectSchedule, bool) {
	project, err := a.queries.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		http.NotFound(w, r)
		return projectSchedule{}, false
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return projectSchedule{}, false
	}

	schedule, err := a.queries.GetSchedule(r.Context(), scheduleID)
	if errors.Is(err, sql.ErrNoRows) || schedule.ProjectID != project.ID {
		http.NotFound(w, r)
		return projectSchedule{}, false
	}
	if err != nil {
		http.Error(w, "could not load schedule", http.StatusInternalServerError)
		return projectSchedule{}, false
	}

	return projectSchedule{project: project, schedule: schedule}, true
}

func (a app) loadProjectScheduleItem(w http.ResponseWriter, r *http.Request) (projectSchedule, queries.ScheduleItem, bool) {
	loaded, ok := a.loadProjectSchedule(w, r, r.PathValue("projectID"), r.PathValue("scheduleID"))
	if !ok {
		return projectSchedule{}, queries.ScheduleItem{}, false
	}

	item, err := a.queries.GetScheduleItem(r.Context(), r.PathValue("itemID"))
	if errors.Is(err, sql.ErrNoRows) || item.ScheduleID != loaded.schedule.ID {
		http.NotFound(w, r)
		return projectSchedule{}, queries.ScheduleItem{}, false
	}
	if err != nil {
		http.Error(w, "could not load item", http.StatusInternalServerError)
		return projectSchedule{}, queries.ScheduleItem{}, false
	}

	return loaded, item, true
}

func (a app) requireUser(w http.ResponseWriter, r *http.Request) (queries.User, bool) {
	if a.queries == nil {
		http.Error(w, "database is unavailable", http.StatusServiceUnavailable)
		return queries.User{}, false
	}

	if a.oauthConfig != nil {
		email, ok := getSessionEmail(r, a.sessionSecret)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return queries.User{}, false
		}
		user, err := a.queries.GetUserByEmail(r.Context(), email)
		if errors.Is(err, sql.ErrNoRows) {
			clearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return queries.User{}, false
		}
		if err != nil {
			http.Error(w, "could not load user", http.StatusInternalServerError)
			return queries.User{}, false
		}
		return user, true
	}

	user, err := a.queries.CreateUser(context.Background(), queries.CreateUserParams{
		Email: a.devUserEmail,
		Name:  a.devUserName,
	})
	if err != nil {
		http.Error(w, "could not load user", http.StatusInternalServerError)
		return queries.User{}, false
	}
	return user, true
}

type projectsPage struct {
	Kind     string
	Title    string
	Projects []queries.Project
}

type projectFormPage struct {
	Kind  string
	Title string
}

type projectDetailPage struct {
	Kind      string
	Title     string
	Project   queries.Project
	Schedules []scheduleWithItems
}

type projectEditPage struct {
	Kind    string
	Title   string
	Project queries.Project
}

type scheduleEditPage struct {
	Kind     string
	Title    string
	Project  queries.Project
	Schedule queries.Schedule
}

type itemEditPage struct {
	Kind     string
	Title    string
	Project  queries.Project
	Schedule queries.Schedule
	Item     queries.ScheduleItem
}

type shareManagePage struct {
	Kind         string
	Title        string
	Project      queries.Project
	ActiveLink   *queries.ProjectShareLink
	ShareBaseURL string
}

type signInPage struct {
	Kind  string
	Title string
}

type publicSharePage struct {
	Kind      string
	Title     string
	Project   queries.Project
	Schedules []scheduleWithItems
}

type scheduleWithItems struct {
	Schedule queries.Schedule
	Items    []queries.ScheduleItem
}

func (a app) schedulesWithItems(ctx context.Context, projectID string) ([]scheduleWithItems, error) {
	schedules, err := a.queries.ListSchedulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]scheduleWithItems, 0, len(schedules))
	for _, schedule := range schedules {
		items, err := a.queries.ListScheduleItems(ctx, schedule.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, scheduleWithItems{Schedule: schedule, Items: items})
	}
	return result, nil
}

func render(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := pageTemplate.Execute(w, data); err != nil {
		http.Error(w, "could not render page", http.StatusInternalServerError)
	}
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func requestBaseURL(r *http.Request) string {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

func splitLines(value string) []string {
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseInt32(value string, fallback int32) int32 {
	var parsed int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed); err != nil {
		return fallback
	}
	return int32(parsed)
}

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - Sally</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
  <header><strong>Sally</strong></header>
  <main>
    {{template "content" .}}
  </main>
</body>
</html>

{{define "content"}}
  {{if eq .Kind "projects"}}
    <h1>Projects</h1>
    <p class="actions"><a class="button" href="/projects/new">New Project</a></p>
    {{if .Projects}}
      <table>
        <thead><tr><th>Name</th><th>Address</th><th>Updated</th></tr></thead>
        <tbody>
          {{range .Projects}}
            <tr>
              <td><a href="/projects/{{.ID}}">{{.Name}}</a></td>
              <td>{{.Address}}</td>
              <td>{{.UpdatedAt.Format "2006-01-02"}}</td>
            </tr>
          {{end}}
        </tbody>
      </table>
    {{else}}
      <p class="muted">No projects yet.</p>
    {{end}}
  {{else if eq .Kind "new-project"}}
    <h1>New Project</h1>
    <form method="post" action="/projects">
      <label>Project Name <input name="name" required></label>
      <label>Address <input name="address"></label>
      <button type="submit">Create Project</button>
    </form>
  {{else if eq .Kind "project"}}
    <p><a href="/projects">Projects</a></p>
    <h1>{{.Project.Name}}</h1>
    {{if .Project.Address}}<p>{{.Project.Address}}</p>{{end}}
    <p class="actions"><a class="button" href="/projects/{{.Project.ID}}/edit">Edit</a> <a class="button" href="/projects/{{.Project.ID}}/share">Share</a></p>
    <form method="post" action="/projects/{{.Project.ID}}/schedules">
      <label>New Schedule <input name="name" required></label>
      <button type="submit">Add Schedule</button>
    </form>
    {{if .Schedules}}
      <div class="project-layout">
        <nav class="schedule-nav">
          <ul>
            {{range .Schedules}}
              <li><a href="#schedule-{{.Schedule.ID}}">{{.Schedule.Name}}</a></li>
            {{end}}
          </ul>
        </nav>
        <div class="schedule-content">
          {{range .Schedules}}
            {{$s := .Schedule}}
            <details class="schedule-section" id="schedule-{{$s.ID}}" open>
              <summary>
                <h2>{{$s.Name}}</h2>
                <span class="summary-toggle"></span>
              </summary>
              <div class="schedule-actions">
                <a href="/projects/{{$.Project.ID}}/schedules/{{$s.ID}}/edit">Edit Schedule</a>
              </div>
              {{if .Items}}
                <table>
                  <thead><tr><th></th><th>Code</th><th>Description</th><th>Manufacturer</th><th>Finish</th><th>Notes</th><th></th></tr></thead>
                  <tbody>
                    {{range .Items}}
                      <tr>
                        <td>{{if .SourceImageUrl}}<img class="item-thumb" src="{{.SourceImageUrl}}" alt="">{{end}}</td>
                        <td>{{.Code}}</td>
                        <td>{{if .SourceUrl}}<a href="{{.SourceUrl}}">{{.Title}}</a>{{else}}{{.Title}}{{end}}<br><span class="muted">{{.Description}}</span></td>
                        <td>{{.Manufacturer}} {{.ModelNumber}}</td>
                        <td>{{.Finish}}</td>
                        <td>{{.Notes}}</td>
                        <td><a href="/projects/{{$.Project.ID}}/schedules/{{$s.ID}}/items/{{.ID}}/edit">Edit</a></td>
                      </tr>
                    {{end}}
                  </tbody>
                </table>
              {{else}}
                <span class="schedule-empty muted">No items yet.</span>
              {{end}}
            </details>
          {{end}}
        </div>
      </div>
    {{else}}
      <p class="muted">No schedules yet.</p>
    {{end}}
  {{else if eq .Kind "edit-project"}}
    <p><a href="/projects/{{.Project.ID}}">{{.Project.Name}}</a></p>
    <h1>Edit Project</h1>
    <form method="post" action="/projects/{{.Project.ID}}/edit">
      <label>Project Name <input name="name" value="{{.Project.Name}}" required></label>
      <label>Address <input name="address" value="{{.Project.Address}}"></label>
      <button type="submit">Update Project</button>
    </form>
    <form method="post" action="/projects/{{.Project.ID}}/delete">
      <button type="submit">Delete Project</button>
    </form>
  {{else if eq .Kind "edit-schedule"}}
    <p><a href="/projects">Projects</a> / <a href="/projects/{{.Project.ID}}">{{.Project.Name}}</a> / <a href="/projects/{{.Project.ID}}#schedule-{{.Schedule.ID}}">{{.Schedule.Name}}</a></p>
    <h1>Edit Schedule</h1>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/edit">
      <label>Schedule Name <input name="name" value="{{.Schedule.Name}}" required></label>
      <label>Position <input name="position" value="{{.Schedule.Position}}"></label>
      <button type="submit">Update Schedule</button>
    </form>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/delete">
      <button type="submit">Delete Schedule</button>
    </form>
  {{else if eq .Kind "edit-item"}}
    <p><a href="/projects">Projects</a> / <a href="/projects/{{.Project.ID}}">{{.Project.Name}}</a> / <a href="/projects/{{.Project.ID}}#schedule-{{.Schedule.ID}}">{{.Schedule.Name}}</a></p>
    <h1>Edit Item</h1>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/items/{{.Item.ID}}/edit">
      <label>Code <input name="code" value="{{.Item.Code}}"></label>
      <label>Title <input name="title" value="{{.Item.Title}}" required></label>
      <label>Description <input name="description" value="{{.Item.Description}}"></label>
      <label>Manufacturer <input name="manufacturer" value="{{.Item.Manufacturer}}"></label>
      <label>Model Number <input name="model_number" value="{{.Item.ModelNumber}}"></label>
      <label>Finish <input name="finish" value="{{.Item.Finish}}"></label>
      <label>Finish Model Number <input name="finish_model_number" value="{{.Item.FinishModelNumber}}"></label>
      <label>Notes <input name="notes" value="{{.Item.Notes}}"></label>
      <label>Source URL <input name="source_url" value="{{.Item.SourceUrl}}"></label>
      <label>Position <input name="position" value="{{.Item.Position}}"></label>
      <button type="submit">Update Item</button>
    </form>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/items/{{.Item.ID}}/delete">
      <button type="submit">Delete Item</button>
    </form>
  {{else if eq .Kind "share-manage"}}
    <p><a href="/projects">Projects</a> / <a href="/projects/{{.Project.ID}}">{{.Project.Name}}</a></p>
    <h1>Share</h1>
    {{if .ActiveLink}}
      <p class="share-status">Sharing is <strong>enabled</strong>. Copy the link below and send it to your contractor or client.</p>
      <div class="share-url-row">
        <input id="share-url-input" class="share-url-input" type="text" readonly value="{{.ShareBaseURL}}/share/{{.ActiveLink.Token}}">
        <button type="button" class="copy-button" onclick="(function(){var i=document.getElementById('share-url-input');i.select();navigator.clipboard.writeText(i.value).catch(function(){document.execCommand('copy')});})()">Copy</button>
      </div>
      <form method="post" action="/projects/{{.Project.ID}}/share-links/deactivate" class="share-action-form">
        <button type="submit" class="button-danger">Disable sharing</button>
      </form>
    {{else}}
      <p class="muted">Share a read-only link with contractors and clients so they can view the schedule.</p>
      <form method="post" action="/projects/{{.Project.ID}}/share-links">
        <button type="submit">Enable sharing</button>
      </form>
    {{end}}
  {{else if eq .Kind "public-share"}}
    <h1>{{.Project.Name}}</h1>
    {{if .Project.Address}}<p>{{.Project.Address}}</p>{{end}}
    {{range .Schedules}}
      <h2>{{.Schedule.Name}}</h2>
      {{if .Items}}
        <table>
          <thead><tr><th></th><th>Code</th><th>Description</th><th>Manufacturer</th><th>Finish</th><th>Notes</th><th>Source</th></tr></thead>
          <tbody>
            {{range .Items}}
              <tr>
                <td>{{if .SourceImageUrl}}<img class="item-thumb" src="{{.SourceImageUrl}}" alt="">{{end}}</td>
                <td>{{.Code}}</td>
                <td>{{if .SourceUrl}}<a href="{{.SourceUrl}}">{{.Title}}</a>{{else}}{{.Title}}{{end}}<br><span class="muted">{{.Description}}</span></td>
                <td>{{.Manufacturer}} {{.ModelNumber}}</td>
                <td>{{.Finish}}</td>
                <td>{{.Notes}}</td>
                <td>{{if .SourceUrl}}<a href="{{.SourceUrl}}">Product Page</a>{{end}}</td>
              </tr>
            {{end}}
          </tbody>
        </table>
      {{else}}
        <p class="muted">No items.</p>
      {{end}}
    {{end}}
  {{else if eq .Kind "login"}}
    <div class="login-page">
      <h1>Sally</h1>
      <p>Sign in to manage your projects and schedules.</p>
      <a class="button" href="/auth/google">Sign in with Google</a>
    </div>
  {{end}}
{{end}}`))
