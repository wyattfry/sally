package httpapi

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
	"sally/server/internal/share"
	"sally/server/internal/web"
)

type mothershipAPI struct {
	queries       *queries.Queries
	devUserEmail  string
	devUserName   string
	sessionSecret []byte
	oauthEnabled  bool
}

func registerMothershipAPI(mux *http.ServeMux, deps web.Deps) {
	if deps.Queries == nil {
		return
	}
	api := mothershipAPI{
		queries:       deps.Queries,
		devUserEmail:  firstNonEmpty(deps.DevUserEmail, "dev@spexxtool.local"),
		devUserName:   firstNonEmpty(deps.DevUserName, "Development User"),
		sessionSecret: deps.SessionSecret,
		oauthEnabled:  deps.OAuthConfig != nil,
	}
	mux.HandleFunc("GET /api/v1/me", api.getMe)
	mux.HandleFunc("GET /api/v1/projects", api.listProjects)
	mux.HandleFunc("POST /api/v1/projects", api.createProject)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/schedules", api.listSchedules)
	mux.HandleFunc("POST /api/v1/projects/{projectID}/schedules", api.createSchedule)
	mux.HandleFunc("GET /api/v1/projects/{projectID}/duplicate-check", api.checkDuplicate)
	mux.HandleFunc("GET /api/v1/schedules/{scheduleID}/columns", api.listScheduleColumns)
	mux.HandleFunc("GET /api/v1/schedules/{scheduleID}/next-code", api.getScheduleNextCode)
	mux.HandleFunc("POST /api/v1/schedules/{scheduleID}/items", api.createScheduleItem)
}

func (api mothershipAPI) listProjects(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	owned, err := api.queries.ListProjectsByOwner(r.Context(), user.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load projects")
		return
	}
	shared, err := api.queries.ListSharedProjects(r.Context(), user.ID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load shared projects")
		return
	}
	seen := make(map[string]bool, len(owned))
	merged := make([]queries.Project, 0, len(owned)+len(shared))
	for _, p := range owned {
		seen[p.ID] = true
		merged = append(merged, p)
	}
	for _, p := range shared {
		if !seen[p.ID] {
			merged = append(merged, p)
		}
	}
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].UpdatedAt.After(merged[j].UpdatedAt)
	})
	resp := make([]projectResponse, 0, len(merged))
	for _, p := range merged {
		resp = append(resp, toProjectResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

type createProjectRequest struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

func (api mothershipAPI) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
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
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	projectID := r.PathValue("projectID")
	project, err := api.queries.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load project")
		return
	}
	if api.oauthEnabled && project.OwnerUserID != user.ID {
		writeJSONError(w, http.StatusNotFound, "project not found")
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
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Preset string `json:"preset"`
}

func (api mothershipAPI) createSchedule(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	projectID := r.PathValue("projectID")
	project, err := api.queries.GetProject(r.Context(), projectID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load project")
		return
	}
	if api.oauthEnabled && project.OwnerUserID != user.ID {
		writeJSONError(w, http.StatusNotFound, "project not found")
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
	if req.Kind == "" {
		req.Kind = "items"
	}

	existingSchedules, err := api.queries.ListSchedulesByProject(r.Context(), projectID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load schedules")
		return
	}
	for _, s := range existingSchedules {
		if strings.EqualFold(s.Name, req.Name) {
			writeJSONError(w, http.StatusConflict, "a schedule with that name already exists")
			return
		}
	}

	schedule, err := api.queries.CreateSchedule(r.Context(), queries.CreateScheduleParams{
		ProjectID: projectID,
		Name:      req.Name,
		Kind:      req.Kind,
		Notes:     "",
		Position:  int32(len(existingSchedules) + 1),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create schedule")
		return
	}

	if req.Kind == "items" {
		preset := req.Preset
		if preset == "" {
			preset, _ = presets.InferByName(req.Name)
		}
		for _, col := range presets.Get(preset) {
			if _, err := api.queries.CreateScheduleColumn(r.Context(), queries.CreateScheduleColumnParams{
				ScheduleID: schedule.ID,
				Key:        col.Key,
				Label:      col.Label,
				Kind:       "text",
				Position:   col.Position,
			}); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "could not seed columns")
				return
			}
		}
	}

	writeJSON(w, http.StatusCreated, toScheduleResponse(schedule))
}

func (api mothershipAPI) listScheduleColumns(w http.ResponseWriter, r *http.Request) {
	scheduleID := r.PathValue("scheduleID")
	if _, err := api.queries.GetSchedule(r.Context(), scheduleID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "schedule not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load schedule")
		return
	}

	cols, err := api.queries.ListScheduleColumns(r.Context(), scheduleID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load columns")
		return
	}
	resp := make([]columnResponse, 0, len(cols))
	for _, col := range cols {
		resp = append(resp, toColumnResponse(col))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (api mothershipAPI) getScheduleNextCode(w http.ResponseWriter, r *http.Request) {
	scheduleID := r.PathValue("scheduleID")
	schedule, err := api.queries.GetSchedule(r.Context(), scheduleID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "schedule not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load schedule")
		return
	}
	items, err := api.queries.ListScheduleItems(r.Context(), scheduleID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load items")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"nextCode": nextCode(items, schedule.Name)})
}

type createScheduleItemRequest struct {
	Data             map[string]string `json:"data"`
	Room             string            `json:"room"`
	SourceURL        string            `json:"sourceUrl"`
	SourceTitle      string            `json:"sourceTitle"`
	SourceImageURL   string            `json:"sourceImageUrl"`
	SourceImageURLs  []string          `json:"sourceImageUrls"`
	SourcePDFLinks   []string          `json:"sourcePdfLinks"`
	ConfirmDuplicate bool              `json:"confirmDuplicate"`
}

type duplicateItemResponse struct {
	Error        string `json:"error"`
	Duplicate    bool   `json:"duplicate"`
	ScheduleID   string `json:"scheduleId"`
	ScheduleName string `json:"scheduleName"`
	ItemCode     string `json:"itemCode"`
}

// normalizeMatchKey lowercases, trims, and strips non-alphanumerics. Used to
// compare manufacturer + model_number when looking for already-spec'd items —
// e.g. "K-3589" and "k3589" should collide.
func normalizeMatchKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (api mothershipAPI) checkDuplicate(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := api.queries.GetProject(r.Context(), projectID); errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusNotFound, "project not found")
		return
	} else if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load project")
		return
	}
	incoming := map[string]string{
		"manufacturer": r.URL.Query().Get("manufacturer"),
		"model_number": r.URL.Query().Get("model_number"),
	}
	dup, found := api.findDuplicateItem(r.Context(), projectID, incoming)
	if !found {
		writeJSON(w, http.StatusOK, duplicateItemResponse{Duplicate: false})
		return
	}
	var dupData map[string]string
	_ = json.Unmarshal(dup.Data, &dupData)
	writeJSON(w, http.StatusOK, duplicateItemResponse{
		Duplicate:    true,
		ScheduleID:   dup.ScheduleID,
		ScheduleName: dup.ScheduleName,
		ItemCode:     dupData["code"],
	})
}

// findDuplicateItem scans every item in the project for one whose
// manufacturer+model_number matches the incoming data. Returns an empty row
// when either key is missing or no match is found.
func (api mothershipAPI) findDuplicateItem(ctx context.Context, projectID string, incoming map[string]string) (queries.ListProjectItemsWithScheduleRow, bool) {
	mfg := normalizeMatchKey(incoming["manufacturer"])
	model := normalizeMatchKey(incoming["model_number"])
	if mfg == "" || model == "" {
		return queries.ListProjectItemsWithScheduleRow{}, false
	}
	items, err := api.queries.ListProjectItemsWithSchedule(ctx, projectID)
	if err != nil {
		return queries.ListProjectItemsWithScheduleRow{}, false
	}
	for _, it := range items {
		var d map[string]string
		if err := json.Unmarshal(it.Data, &d); err != nil {
			continue
		}
		if normalizeMatchKey(d["manufacturer"]) == mfg && normalizeMatchKey(d["model_number"]) == model {
			return it, true
		}
	}
	return queries.ListProjectItemsWithScheduleRow{}, false
}

func (api mothershipAPI) createScheduleItem(w http.ResponseWriter, r *http.Request) {
	scheduleID := r.PathValue("scheduleID")
	schedule, err := api.queries.GetSchedule(r.Context(), scheduleID)
	if errors.Is(err, sql.ErrNoRows) {
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
	if req.Data == nil {
		req.Data = map[string]string{}
	}
	if req.SourcePDFLinks == nil {
		req.SourcePDFLinks = []string{}
	}
	if req.SourceImageURLs == nil {
		req.SourceImageURLs = []string{}
	}

	existingItems, err := api.queries.ListScheduleItems(r.Context(), scheduleID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load items")
		return
	}

	if !req.ConfirmDuplicate {
		if dup, found := api.findDuplicateItem(r.Context(), schedule.ProjectID, req.Data); found {
			var dupData map[string]string
			_ = json.Unmarshal(dup.Data, &dupData)
			writeJSON(w, http.StatusConflict, duplicateItemResponse{
				Error:        "duplicate item",
				Duplicate:    true,
				ScheduleID:   dup.ScheduleID,
				ScheduleName: dup.ScheduleName,
				ItemCode:     dupData["code"],
			})
			return
		}
	}

	if req.Data["code"] == "" {
		req.Data["code"] = nextCode(existingItems, schedule.Name)
	}

	dataJSON, err := json.Marshal(req.Data)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not encode data")
		return
	}

	item, err := api.queries.CreateScheduleItem(r.Context(), queries.CreateScheduleItemParams{
		ScheduleID:      scheduleID,
		Data:            dataJSON,
		Room:            strings.TrimSpace(req.Room),
		SourceUrl:       strings.TrimSpace(req.SourceURL),
		SourceTitle:     strings.TrimSpace(req.SourceTitle),
		SourceImageUrl:  strings.TrimSpace(req.SourceImageURL),
		SourceImageUrls: req.SourceImageURLs,
		SourcePdfLinks:  req.SourcePDFLinks,
		Position:        int32(len(existingItems) + 1),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create item")
		return
	}

	// Paint extractions populate data.color even when the schedule doesn't
	// yet have a Color column — add the column on first such save so the
	// value becomes visible rather than orphaned in the data jsonb.
	if strings.TrimSpace(req.Data["color"]) != "" {
		api.ensureColumn(r.Context(), scheduleID, "color", "Color")
	}

	writeJSON(w, http.StatusCreated, toScheduleItemResponse(item))
}

func (api mothershipAPI) ensureColumn(ctx context.Context, scheduleID, key, label string) {
	cols, err := api.queries.ListScheduleColumns(ctx, scheduleID)
	if err != nil {
		return
	}
	maxPos := int32(0)
	for _, c := range cols {
		if c.Key == key {
			return
		}
		if c.Position > maxPos {
			maxPos = c.Position
		}
	}
	_, _ = api.queries.CreateScheduleColumn(ctx, queries.CreateScheduleColumnParams{
		ScheduleID: scheduleID,
		Key:        key,
		Label:      label,
		Kind:       "text",
		Position:   maxPos + 1,
	})
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
	Kind      string `json:"kind"`
	Notes     string `json:"notes"`
	Position  int32  `json:"position"`
}

type columnResponse struct {
	ID         string `json:"id"`
	ScheduleID string `json:"scheduleId"`
	Key        string `json:"key"`
	Label      string `json:"label"`
	Kind       string `json:"kind"`
	Position   int32  `json:"position"`
}

type scheduleItemResponse struct {
	ID              string            `json:"id"`
	ScheduleID      string            `json:"scheduleId"`
	Data            map[string]string `json:"data"`
	Room            string            `json:"room"`
	SourceURL       string            `json:"sourceUrl"`
	SourceTitle     string            `json:"sourceTitle"`
	SourceImageURL  string            `json:"sourceImageUrl"`
	SourceImageURLs []string          `json:"sourceImageUrls"`
	SourcePDFLinks  []string          `json:"sourcePdfLinks"`
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
		Kind:      schedule.Kind,
		Notes:     schedule.Notes,
		Position:  schedule.Position,
	}
}

func toColumnResponse(col queries.ScheduleColumn) columnResponse {
	return columnResponse{
		ID:         col.ID,
		ScheduleID: col.ScheduleID,
		Key:        col.Key,
		Label:      col.Label,
		Kind:       col.Kind,
		Position:   col.Position,
	}
}

func toScheduleItemResponse(item queries.ScheduleItem) scheduleItemResponse {
	var data map[string]string
	if len(item.Data) > 0 {
		_ = json.Unmarshal(item.Data, &data)
	}
	if data == nil {
		data = map[string]string{}
	}
	links := item.SourcePdfLinks
	if links == nil {
		links = []string{}
	}
	imgURLs := item.SourceImageUrls
	if imgURLs == nil {
		imgURLs = []string{}
	}
	return scheduleItemResponse{
		ID:              item.ID,
		ScheduleID:      item.ScheduleID,
		Data:            data,
		Room:            item.Room,
		SourceURL:       item.SourceUrl,
		SourceTitle:     item.SourceTitle,
		SourceImageURL:  item.SourceImageUrl,
		SourceImageURLs: imgURLs,
		SourcePDFLinks:  links,
	}
}

func (api mothershipAPI) requireUser(w http.ResponseWriter, r *http.Request) (queries.User, bool) {
	// API token via Authorization: Bearer <token> — works regardless of oauth mode.
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		raw := strings.TrimPrefix(auth, "Bearer ")
		hash := share.HashToken(raw)
		apiToken, err := api.queries.GetAPITokenByHash(r.Context(), hash)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "invalid api token")
			return queries.User{}, false
		}
		user, err := api.queries.GetUser(r.Context(), apiToken.UserID)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return queries.User{}, false
		}
		// Update last_used_at in the background so we don't block the request.
		go func() { _ = api.queries.TouchAPITokenLastUsed(r.Context(), apiToken.ID) }()
		return user, true
	}

	if api.oauthEnabled {
		var email string
		var ok bool
		if token := r.Header.Get("X-Session-Token"); token != "" {
			email, ok = web.ValidateSessionToken(api.sessionSecret, token)
		} else {
			email, ok = web.GetSessionEmail(r, api.sessionSecret)
		}
		if !ok {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return queries.User{}, false
		}
		user, err := api.queries.GetUserByEmail(r.Context(), email)
		if err != nil {
			writeJSONError(w, http.StatusUnauthorized, "unauthorized")
			return queries.User{}, false
		}
		return user, true
	}
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

type meResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (api mothershipAPI) getMe(w http.ResponseWriter, r *http.Request) {
	user, ok := api.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, meResponse{ID: user.ID, Name: user.Name, Email: user.Email})
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
