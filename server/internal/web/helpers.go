package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/presets"
)

type projectSchedule struct {
	project  queries.Project
	schedule queries.Schedule
}

// loadUserProject allows access for the project owner or any editor member.
func (a app) loadUserProject(w http.ResponseWriter, r *http.Request, projectID string) (queries.User, queries.Project, bool) {
	user, ok := a.requireUser(w, r)
	if !ok {
		return queries.User{}, queries.Project{}, false
	}
	project, err := a.queries.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		renderNotFound(w)
		return queries.User{}, queries.Project{}, false
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return queries.User{}, queries.Project{}, false
	}
	if a.oauthConfig != nil && project.OwnerUserID != user.ID {
		_, memberErr := a.queries.GetProjectMember(r.Context(), queries.GetProjectMemberParams{
			ProjectID: projectID,
			UserID:    user.ID,
		})
		if errors.Is(memberErr, sql.ErrNoRows) {
			renderNotFound(w)
			return queries.User{}, queries.Project{}, false
		}
		if memberErr != nil {
			http.Error(w, "could not verify access", http.StatusInternalServerError)
			return queries.User{}, queries.Project{}, false
		}
	}
	return user, project, true
}

// loadUserProjectAsOwner requires the current user to be the project owner.
// Use this for destructive or administrative operations.
func (a app) loadUserProjectAsOwner(w http.ResponseWriter, r *http.Request, projectID string) (queries.User, queries.Project, bool) {
	user, ok := a.requireUser(w, r)
	if !ok {
		return queries.User{}, queries.Project{}, false
	}
	project, err := a.queries.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		renderNotFound(w)
		return queries.User{}, queries.Project{}, false
	}
	if err != nil {
		http.Error(w, "could not load project", http.StatusInternalServerError)
		return queries.User{}, queries.Project{}, false
	}
	if a.oauthConfig != nil && project.OwnerUserID != user.ID {
		renderNotFound(w)
		return queries.User{}, queries.Project{}, false
	}
	return user, project, true
}

func (a app) loadProjectSchedule(w http.ResponseWriter, r *http.Request, projectID string, scheduleID string) (projectSchedule, bool) {
	_, project, ok := a.loadUserProject(w, r, projectID)
	if !ok {
		return projectSchedule{}, false
	}

	schedule, err := a.queries.GetSchedule(r.Context(), scheduleID)
	if errors.Is(err, sql.ErrNoRows) || schedule.ProjectID != project.ID {
		renderNotFound(w)
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
		renderNotFound(w)
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
		if col.Key == "zone" {
			continue // zone has its own DB column; not stored in data JSONB
		}
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
	dm["zone"] = item.Zone
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
		// Auto-sort by code if a "code" column exists.
		for _, col := range cols {
			if col.Key == "code" {
				sort.Slice(views, func(i, j int) bool {
					return strings.ToLower(views[i].DataMap["code"]) < strings.ToLower(views[j].DataMap["code"])
				})
				break
			}
		}
		result = append(result, scheduleWithItems{
			Schedule: schedule,
			Columns:  cols,
			Groups:   groupByZone(views),
		})
	}
	return result, nil
}
