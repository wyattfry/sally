package web

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
)

type Deps struct {
	Queries      *queries.Queries
	DevUserEmail string
	DevUserName  string
}

type app struct {
	queries      *queries.Queries
	devUserEmail string
	devUserName  string
}

func RegisterRoutes(mux *http.ServeMux, deps Deps) {
	a := app{
		queries:      deps.Queries,
		devUserEmail: firstNonEmpty(deps.DevUserEmail, "dev@spexxtool.local"),
		devUserName:  firstNonEmpty(deps.DevUserName, "Development User"),
	}

	mux.HandleFunc("GET /", a.redirectHome)
	mux.HandleFunc("GET /projects", a.listProjects)
	mux.HandleFunc("GET /projects/new", a.newProject)
	mux.HandleFunc("POST /projects", a.createProject)
	mux.HandleFunc("GET /projects/{projectID}", a.showProject)
}

func (a app) redirectHome(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (a app) listProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := a.requireDevUser(w, r)
	if !ok {
		return
	}

	projects, err := a.queries.ListProjectsByOwner(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "could not load projects", http.StatusInternalServerError)
		return
	}

	render(w, projectsPage{
		Title:    "Projects",
		Projects: projects,
	})
}

func (a app) newProject(w http.ResponseWriter, _ *http.Request) {
	render(w, projectFormPage{Title: "New Project"})
}

func (a app) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := a.requireDevUser(w, r)
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

	schedules, err := a.queries.ListSchedulesByProject(r.Context(), project.ID)
	if err != nil {
		http.Error(w, "could not load schedules", http.StatusInternalServerError)
		return
	}

	render(w, projectDetailPage{
		Title:     project.Name,
		Project:   project,
		Schedules: schedules,
	})
}

func (a app) requireDevUser(w http.ResponseWriter, r *http.Request) (queries.User, bool) {
	if a.queries == nil {
		http.Error(w, "database is unavailable", http.StatusServiceUnavailable)
		return queries.User{}, false
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
	Title    string
	Projects []queries.Project
}

type projectFormPage struct {
	Title string
}

type projectDetailPage struct {
	Title     string
	Project   queries.Project
	Schedules []queries.Schedule
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

var pageTemplate = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} - Sally</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 0; color: #161616; background: #f7f7f4; }
    header { background: #0f2f24; color: white; padding: 14px 24px; }
    main { max-width: 1120px; margin: 0 auto; padding: 24px; }
    a { color: #145f43; }
    table { width: 100%; border-collapse: collapse; background: white; }
    th, td { border-bottom: 1px solid #d8d8d2; padding: 10px; text-align: left; vertical-align: top; }
    th { font-size: 12px; text-transform: uppercase; letter-spacing: .03em; background: #ecece6; }
    input { box-sizing: border-box; width: 100%; padding: 8px; border: 1px solid #aaa; }
    label { display: block; margin: 12px 0; font-weight: 700; }
    button, .button { display: inline-block; padding: 8px 12px; border: 1px solid #145f43; background: #b7f3c5; color: #102015; text-decoration: none; border-radius: 4px; cursor: pointer; }
    .actions { margin: 16px 0; }
    .muted { color: #666; }
  </style>
</head>
<body>
  <header><strong>Sally</strong></header>
  <main>
    {{template "content" .}}
  </main>
</body>
</html>

{{define "content"}}
  {{if eq .Title "Projects"}}
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
  {{else if eq .Title "New Project"}}
    <h1>New Project</h1>
    <form method="post" action="/projects">
      <label>Project Name <input name="name" required></label>
      <label>Address <input name="address"></label>
      <button type="submit">Create Project</button>
    </form>
  {{else}}
    <p><a href="/projects">Projects</a></p>
    <h1>{{.Project.Name}}</h1>
    {{if .Project.Address}}<p>{{.Project.Address}}</p>{{end}}
    <h2>Schedules</h2>
    {{if .Schedules}}
      <table>
        <thead><tr><th>Name</th><th>Updated</th></tr></thead>
        <tbody>
          {{range .Schedules}}
            <tr><td>{{.Name}}</td><td>{{.UpdatedAt.Format "2006-01-02"}}</td></tr>
          {{end}}
        </tbody>
      </table>
    {{else}}
      <p class="muted">No schedules yet.</p>
    {{end}}
  {{end}}
{{end}}`))
