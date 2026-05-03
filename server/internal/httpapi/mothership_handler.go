package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/web"
)

type mothershipAPI struct {
	queries      *queries.Queries
	devUserEmail string
	devUserName  string
}

func registerMothershipAPI(mux *http.ServeMux, deps web.Deps) {
	if deps.Queries == nil {
		return
	}
	api := mothershipAPI{
		queries:      deps.Queries,
		devUserEmail: firstNonEmpty(deps.DevUserEmail, "dev@spexxtool.local"),
		devUserName:  firstNonEmpty(deps.DevUserName, "Development User"),
	}
	mux.HandleFunc("GET /api/v1/projects", api.listProjects)
	mux.HandleFunc("POST /api/v1/projects", api.createProject)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/schedules", api.listSchedules)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/schedules", api.createSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{scheduleID}/items", api.createScheduleItem)
}

func (api mothershipAPI) listProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireDevUser(w, r)
	if !ok {
		return
	}
	projects, err := api.queries.ListProjectsByOwner(r.Context(), user.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load projects")
		return
	}
	resp := make([]projectResponse, 0, len(projects))
	for _, project := range projects {
		resp = append(resp, toProjectResponse(project))
	}
	writeJSON(w, http.StatusOK, resp)
}

type createProjectRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (api mothershipAPI) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireDevUser(w, r)
	if !ok {
		return
	}

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "project name is required")
		return
	}

	project, err := api.queries.CreateProject(r.Context(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        req.Name,
		Address:     req.Address,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create project")
		return
	}

	writeJSON(w, http.StatusCreated, toProjectResponse(project))
}

func (api mothershipAPI) listSchedules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := api.queries.GetProject(r.Context(), projectID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load project")
		return
	}

	schedules, err := api.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load schedules")
		return
	}
	resp := make([]scheduleResponse, 0, len(schedules))
	for _, schedule := range schedules {
		resp = append(resp, toScheduleResponse(schedule))
	}
	writeJSON(w, http.StatusOK, resp)
}

type createScheduleRequest struct {
	Name string `json:"name"`
}

func (api mothershipAPI) createSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := api.queries.GetProject(r.Context(), projectID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load project")
		return
	}

	var req createScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeJSONError(w, http.StatusBadRequest, "schedule name is required")
		return
	}

	existingSchedules, err := api.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load schedules")
		return
	}

	schedule, err := api.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: projectID,
		Name:      req.Name,
		Position:  int32(len(existingSchedules) + 1),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create schedule")
		return
	}

	writeJSON(w, http.StatusCreated, toScheduleResponse(schedule))
}

type createScheduleItemRequest struct {
	Code              string   `json:"code"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Manufacturer      string   `json:"manufacturer"`
	ModelNumber       string   `json:"modelNumber"`
	Finish            string   `json:"finish"`
	FinishModelNumber string   `json:"finishModelNumber"`
	Notes             string   `json:"notes"`
	SourceURL         string   `json:"sourceUrl"`
	SourceTitle       string   `json:"sourceTitle"`
	SourceImageURL    string   `json:"sourceImageUrl"`
	SourcePDFLinks    []string `json:"sourcePdfLinks"`
}

func (api mothershipAPI) createScheduleItem(w http.ResponseWriter, r *http.Request) {
	scheduleID := r.PathValue("scheduleID")
	if _, err := api.queries.GetSchedule(r.Context(), scheduleID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "schedule not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load schedule")
		return
	}

	var req createScheduleItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json")
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "item title is required")
		return
	}

	existingItems, err := api.queries.ListScheduleItems(r.Context(), scheduleID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load items")
		return
	}
	item, err := api.queries.CreateScheduleItem(r.Context(), queries.CreateScheduleItemParams{
		ScheduleID:        scheduleID,
		Code:              strings.TrimSpace(req.Code),
		Title:             req.Title,
		Description:       strings.TrimSpace(req.Description),
		Manufacturer:      strings.TrimSpace(req.Manufacturer),
		ModelNumber:       strings.TrimSpace(req.ModelNumber),
		Finish:            strings.TrimSpace(req.Finish),
		FinishModelNumber: strings.TrimSpace(req.FinishModelNumber),
		Notes:             strings.TrimSpace(req.Notes),
		SourceUrl:         strings.TrimSpace(req.SourceURL),
		SourceTitle:       strings.TrimSpace(req.SourceTitle),
		SourceImageUrl:    strings.TrimSpace(req.SourceImageURL),
		SourcePdfLinks:    req.SourcePDFLinks,
		Position:          int32(len(existingItems) + 1),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create item")
		return
	}
	writeJSON(w, http.StatusCreated, toScheduleItemResponse(item))
}

type projectResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updatedAt"`
}

type scheduleResponse struct {
	ID        string `json:"id"`
	ProjectID string `json:"projectId"`
	Name      string `json:"name"`
	Position  int32  `json:"position"`
}

type scheduleItemResponse struct {
	ID                string   `json:"id"`
	ScheduleID        string   `json:"scheduleId"`
	Code              string   `json:"code"`
	Title             string   `json:"title"`
	Description       string   `json:"description"`
	Manufacturer      string   `json:"manufacturer"`
	ModelNumber       string   `json:"modelNumber"`
	Finish            string   `json:"finish"`
	FinishModelNumber string   `json:"finishModelNumber"`
	Notes             string   `json:"notes"`
	SourceURL         string   `json:"sourceUrl"`
	SourceTitle       string   `json:"sourceTitle"`
	SourceImageURL    string   `json:"sourceImageUrl"`
	SourcePDFLinks    []string `json:"sourcePdfLinks"`
}

func toProjectResponse(project queries.Project) projectResponse {
	return projectResponse{
		ID:          project.ID,
		Name:        project.Name,
		Address:     project.Address,
		Description: project.Description,
		UpdatedAt:   project.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func toScheduleResponse(schedule queries.Schedule) scheduleResponse {
	return scheduleResponse{
		ID:        schedule.ID,
		ProjectID: schedule.ProjectID,
		Name:      schedule.Name,
		Position:  schedule.Position,
	}
}

func toScheduleItemResponse(item queries.ScheduleItem) scheduleItemResponse {
	return scheduleItemResponse{
		ID:                item.ID,
		ScheduleID:        item.ScheduleID,
		Code:              item.Code,
		Title:             item.Title,
		Description:       item.Description,
		Manufacturer:      item.Manufacturer,
		ModelNumber:       item.ModelNumber,
		Finish:            item.Finish,
		FinishModelNumber: item.FinishModelNumber,
		Notes:             item.Notes,
		SourceURL:         item.SourceUrl,
		SourceTitle:       item.SourceTitle,
		SourceImageURL:    item.SourceImageUrl,
		SourcePDFLinks:    item.SourcePdfLinks,
	}
}

func (api mothershipAPI) requireDevUser(w http.ResponseWriter, r *http.Request) (queries.User, bool) {
	user, err := api.queries.CreateUser(r.Context(), queries.CreateUserParams{
		Email: api.devUserEmail,
		Name:  api.devUserName,
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load user")
		return queries.User{}, false
	}
	return user, true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func firstNonEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
