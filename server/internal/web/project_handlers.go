package web

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
	queries "sally/server/internal/db/generated"
	"sally/server/internal/presets"
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
		Description: strings.TrimSpace(r.Form.Get("description")),
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

	render(w, projectDetailPage{
		Kind:           "project",
		Title:          project.Name,
		Project:        project,
		Schedules:      schedules,
		FirstItemImage: firstItemImage,
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
		Zone:           strings.TrimSpace(r.Form.Get("zone")),
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
		Zone:           strings.TrimSpace(r.Form.Get("zone")),
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

// ── helpers ──────────────────────────────────────────────────────────────────

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

// seedColumns inserts preset column definitions for a newly created schedule.
func seedColumns(ctx context.Context, q *queries.Queries, scheduleID, presetName string) error {
	for _, col := range presets.Get(presetName) {
		_, err := q.CreateScheduleColumn(ctx, queries.CreateScheduleColumnParams{
			ScheduleID: scheduleID,
			Key:        col.Key,
			Label:      col.Label,
			Kind:       "text",
			Position:   col.Position,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// buildDataMap reads col_<key> form fields for each column and returns a data map.
func buildDataMap(r *http.Request, columns []queries.ScheduleColumn) map[string]string {
	data := make(map[string]string, len(columns))
	for _, col := range columns {
		if v := strings.TrimSpace(r.Form.Get("col_" + col.Key)); v != "" {
			data[col.Key] = v
		}
	}
	return data
}

// scheduleItemView wraps ScheduleItem with a pre-parsed data map for templates.
type scheduleItemView struct {
	queries.ScheduleItem
	DataMap map[string]string
}

func toItemView(item queries.ScheduleItem) scheduleItemView {
	var dm map[string]string
	_ = json.Unmarshal(item.Data, &dm)
	if dm == nil {
		dm = map[string]string{}
	}
	return scheduleItemView{ScheduleItem: item, DataMap: dm}
}

func itemDisplayTitle(item queries.ScheduleItem) string {
	var dm map[string]string
	_ = json.Unmarshal(item.Data, &dm)
	if t := dm["title"]; t != "" {
		return t
	}
	if t := dm["description"]; t != "" {
		return t
	}
	return "item"
}

type zoneGroup struct {
	Zone  string
	Items []scheduleItemView
}

type scheduleWithItems struct {
	Schedule queries.Schedule
	Columns  []queries.ScheduleColumn
	Groups   []zoneGroup
}

func groupByZone(items []scheduleItemView) []zoneGroup {
	seen := map[string]int{}
	var groups []zoneGroup
	for _, item := range items {
		if idx, ok := seen[item.Zone]; ok {
			groups[idx].Items = append(groups[idx].Items, item)
		} else {
			seen[item.Zone] = len(groups)
			groups = append(groups, zoneGroup{Zone: item.Zone, Items: []scheduleItemView{item}})
		}
	}
	return groups
}

func (a app) schedulesWithItems(ctx context.Context, projectID string) ([]scheduleWithItems, error) {
	schedules, err := a.queries.ListSchedulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]scheduleWithItems, 0, len(schedules))
	for _, schedule := range schedules {
		cols, err := a.queries.ListScheduleColumns(ctx, schedule.ID)
		if err != nil {
			return nil, err
		}
		rawItems, err := a.queries.ListScheduleItems(ctx, schedule.ID)
		if err != nil {
			return nil, err
		}
		views := make([]scheduleItemView, len(rawItems))
		for i, it := range rawItems {
			views[i] = toItemView(it)
		}
		result = append(result, scheduleWithItems{
			Schedule: schedule,
			Columns:  cols,
			Groups:   groupByZone(views),
		})
	}
	return result, nil
}

// ── page data types ───────────────────────────────────────────────────────────

type projectListItem struct {
	Project     queries.Project
	ThumbImages []string
}

type projectsPage struct {
	Kind     string
	Title    string
	Projects []projectListItem
}

type projectFormPage struct {
	Kind  string
	Title string
}

type projectDetailPage struct {
	Kind           string
	Title          string
	Project        queries.Project
	Schedules      []scheduleWithItems
	FirstItemImage string
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
	Item     scheduleItemView
	Columns  []queries.ScheduleColumn
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

// ── utilities ─────────────────────────────────────────────────────────────────

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

// ── template ──────────────────────────────────────────────────────────────────

var pageTemplate = template.Must(template.New("page").Funcs(template.FuncMap{
	"add": func(a, b int) int { return a + b },
}).Parse(`<!doctype html>
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
      <div class="project-cards">
        {{range .Projects}}
          <a class="project-card" href="/projects/{{.Project.ID}}">
            <div class="project-thumb-wrap">
              {{if .Project.ThumbnailUrl}}
                <img src="{{.Project.ThumbnailUrl}}" alt="">
              {{else if index .ThumbImages 0}}
                <div class="project-thumb-grid">
                  {{range .ThumbImages}}
                    {{if .}}<img src="{{.}}" alt="">{{else}}<div class="project-thumb-cell"></div>{{end}}
                  {{end}}
                </div>
              {{else}}
                <div class="project-thumb-initial">{{printf "%.1s" .Project.Name}}</div>
              {{end}}
            </div>
            <div class="project-card-body">
              <h2>{{.Project.Name}}</h2>
              {{if .Project.Address}}<p class="project-card-meta">{{.Project.Address}}</p>{{end}}
              {{if .Project.Description}}<p class="project-card-description">{{.Project.Description}}</p>{{end}}
              <p class="project-card-meta">Updated {{.Project.UpdatedAt.Format "2006-01-02"}}</p>
            </div>
          </a>
        {{end}}
      </div>
    {{else}}
      <p class="muted">No projects yet.</p>
    {{end}}
  {{else if eq .Kind "new-project"}}
    <h1>New Project</h1>
    <form method="post" action="/projects">
      <label>Project Name <input name="name" required></label>
      <label>Address <input name="address"></label>
      <label>Description <input name="description"></label>
      <button type="submit">Create Project</button>
    </form>
  {{else if eq .Kind "project"}}
    <p><a href="/projects">Projects</a></p>
    {{if .Project.ThumbnailUrl}}<img src="{{.Project.ThumbnailUrl}}" alt="" style="max-width:320px;display:block;margin-bottom:12px;border-radius:4px;">
    {{else if .FirstItemImage}}<img src="{{.FirstItemImage}}" alt="" style="max-width:320px;display:block;margin-bottom:12px;border-radius:4px;">{{end}}
    <h1>{{.Project.Name}}</h1>
    {{if .Project.Address}}<p class="muted">{{.Project.Address}}</p>{{end}}
    {{if .Project.Description}}<p>{{.Project.Description}}</p>{{end}}
    <p class="actions"><a class="button" href="/projects/{{.Project.ID}}/edit">Edit</a> <a class="button" href="/projects/{{.Project.ID}}/share">Share</a></p>
    <form method="post" action="/projects/{{.Project.ID}}/schedules">
      <label>New Schedule <input name="name" required></label>
      <label>Notes <textarea name="notes" rows="2"></textarea></label>
      <label>Type
        <select name="preset">
          <option value="general">General</option>
          <option value="appliance">Appliance</option>
          <option value="door">Door</option>
          <option value="door_hardware">Door Hardware</option>
          <option value="electrical_fixture">Electrical Fixture</option>
          <option value="insulation">Insulation</option>
          <option value="paint">Paint</option>
          <option value="specialties">Specialties</option>
          <option value="window">Window</option>
        </select>
      </label>
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
            {{$sw := .}}
            <details class="schedule-section" id="schedule-{{$sw.Schedule.ID}}" open>
              <summary>
                <h2>{{$sw.Schedule.Name}}</h2>
                <span class="summary-toggle"></span>
              </summary>
              <div class="schedule-actions">
                <a href="/projects/{{$.Project.ID}}/schedules/{{$sw.Schedule.ID}}/edit">Edit Schedule</a>
              </div>
              {{if $sw.Schedule.Notes}}<p class="schedule-notes">{{$sw.Schedule.Notes}}</p>{{end}}
              {{if eq $sw.Schedule.Kind "note"}}
                <div class="schedule-note-content">{{$sw.Schedule.Notes}}</div>
              {{else if $sw.Groups}}
                <table>
                  <thead>
                    <tr>
                      <th></th>
                      {{range $sw.Columns}}<th>{{.Label}}</th>{{end}}
                      <th></th>
                    </tr>
                  </thead>
                  <tbody>
                    {{range $sw.Groups}}
                      {{$g := .}}
                      {{if $g.Zone}}<tr class="zone-row"><td colspan="{{len $sw.Columns | add 2}}">{{$g.Zone}}</td></tr>{{end}}
                      {{range $g.Items}}
                        {{$item := .}}
                        <tr>
                          <td>{{if $item.SourceImageUrl}}<img class="item-thumb" src="{{$item.SourceImageUrl}}" alt="">{{end}}</td>
                          {{range $sw.Columns}}
                            <td>
                              {{if and (eq .Key "title") $item.SourceUrl}}
                                <a href="{{$item.SourceUrl}}">{{index $item.DataMap .Key}}</a>
                              {{else}}
                                {{index $item.DataMap .Key}}
                              {{end}}
                            </td>
                          {{end}}
                          <td><a href="/projects/{{$.Project.ID}}/schedules/{{$sw.Schedule.ID}}/items/{{$item.ID}}/edit">Edit</a></td>
                        </tr>
                      {{end}}
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
      <label>Description <input name="description" value="{{.Project.Description}}"></label>
      <label>Thumbnail URL <input name="thumbnail_url" value="{{.Project.ThumbnailUrl}}" placeholder="https://… (leave blank to use item images)"></label>
      <button type="submit">Update Project</button>
    </form>
    <form method="post" action="/projects/{{.Project.ID}}/delete">
      <button type="submit">Delete Project</button>
    </form>
  {{else if eq .Kind "edit-schedule"}}
    <p><a href="/projects">Projects</a> / <a href="/projects/{{.Project.ID}}">{{.Project.Name}}</a> / <a href="/projects/{{.Project.ID}}#schedule-{{.Schedule.ID}}">{{.Schedule.Name}}</a></p>
    <h1>Edit Schedule</h1>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/edit">
      <input type="hidden" name="position" value="{{.Schedule.Position}}">
      <input type="hidden" name="kind" value="{{.Schedule.Kind}}">
      <label>Schedule Name <input name="name" value="{{.Schedule.Name}}" required></label>
      <label>Notes <textarea name="notes" rows="3">{{.Schedule.Notes}}</textarea></label>
      <button type="submit">Update Schedule</button>
    </form>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/delete">
      <button type="submit">Delete Schedule</button>
    </form>
  {{else if eq .Kind "edit-item"}}
    <p><a href="/projects">Projects</a> / <a href="/projects/{{.Project.ID}}">{{.Project.Name}}</a> / <a href="/projects/{{.Project.ID}}#schedule-{{.Schedule.ID}}">{{.Schedule.Name}}</a></p>
    <h1>Edit Item</h1>
    <form method="post" action="/projects/{{.Project.ID}}/schedules/{{.Schedule.ID}}/items/{{.Item.ID}}/edit">
      <input type="hidden" name="position" value="{{.Item.Position}}">
      <input type="hidden" name="source_title" value="{{.Item.SourceTitle}}">
      <input type="hidden" name="source_image_url" value="{{.Item.SourceImageUrl}}">
      <textarea name="source_pdf_links" style="display:none">{{range .Item.SourcePdfLinks}}{{.}}
{{end}}</textarea>
      <label>Zone <input name="zone" value="{{.Item.Zone}}" placeholder="e.g. Kitchen, Primary Bath"></label>
      {{range .Columns}}
        <label>{{.Label}} <input name="col_{{.Key}}" value="{{index $.Item.DataMap .Key}}"></label>
      {{end}}
      <label>Source URL <input name="source_url" value="{{.Item.SourceUrl}}"></label>
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
      {{$sw := .}}
      <h2>{{$sw.Schedule.Name}}</h2>
      {{if eq $sw.Schedule.Kind "note"}}
        <div class="schedule-note-content">{{$sw.Schedule.Notes}}</div>
      {{else if $sw.Groups}}
        <table>
          <thead>
            <tr>
              <th></th>
              {{range $sw.Columns}}<th>{{.Label}}</th>{{end}}
              <th>Source</th>
            </tr>
          </thead>
          <tbody>
            {{range $sw.Groups}}
              {{$g := .}}
              {{if $g.Zone}}<tr class="zone-row"><td colspan="{{len $sw.Columns | add 2}}">{{$g.Zone}}</td></tr>{{end}}
              {{range $g.Items}}
                {{$item := .}}
                <tr>
                  <td>{{if $item.SourceImageUrl}}<img class="item-thumb" src="{{$item.SourceImageUrl}}" alt="">{{end}}</td>
                  {{range $sw.Columns}}
                    <td>
                      {{if and (eq .Key "title") $item.SourceUrl}}
                        <a href="{{$item.SourceUrl}}">{{index $item.DataMap .Key}}</a>
                      {{else}}
                        {{index $item.DataMap .Key}}
                      {{end}}
                    </td>
                  {{end}}
                  <td>{{if $item.SourceUrl}}<a href="{{$item.SourceUrl}}">Product Page</a>{{end}}</td>
                </tr>
              {{end}}
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
