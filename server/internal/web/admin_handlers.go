package web

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	appdb "sally/server/internal/db"
	dbgen "sally/server/internal/db/generated"
	"sally/server/internal/share"
)

// requireAdmin checks admin access and returns false (already responded) if denied.
func (a app) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		if a.db == nil || a.queries == nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return false
		}
		raw := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		if raw == "" {
			http.Error(w, "invalid api token", http.StatusUnauthorized)
			return false
		}
		apiToken, err := a.queries.GetAPITokenByHash(r.Context(), share.HashToken(raw))
		if err != nil {
			http.Error(w, "invalid api token", http.StatusUnauthorized)
			return false
		}
		user, err := a.queries.GetUser(r.Context(), apiToken.UserID)
		if err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return false
		}
		if a.oauthConfig != nil && (a.adminEmail == "" || user.Email != a.adminEmail) {
			a.renderNotFound(w, r)
			return false
		}
		_ = a.queries.TouchAPITokenLastUsed(r.Context(), apiToken.ID)
		return true
	}
	if a.oauthConfig != nil {
		user, ok := a.requireUser(w, r)
		if !ok {
			return false
		}
		if a.adminEmail == "" || user.Email != a.adminEmail {
			a.renderNotFound(w, r)
			return false
		}
	}
	if a.db == nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return false
	}
	return true
}

func (a app) adminDashboard(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	ctx := r.Context()

	counts, err := appdb.QueryAdminTableCounts(ctx, a.db)
	if err != nil {
		http.Error(w, "could not load counts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	extractSummary, err := appdb.QueryExtractionSummary(ctx, a.db)
	if err != nil {
		http.Error(w, "could not load extraction summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	providerStats, err := appdb.QueryExtractionProviderStats(ctx, a.db)
	if err != nil {
		http.Error(w, "could not load provider stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	itemDaily, err := appdb.QueryDailyItemSeries(ctx, a.db, 7)
	if err != nil {
		http.Error(w, "could not load item series: "+err.Error(), http.StatusInternalServerError)
		return
	}
	itemHourly, err := appdb.QueryHourlyItemSeries(ctx, a.db, 24)
	if err != nil {
		http.Error(w, "could not load item hourly: "+err.Error(), http.StatusInternalServerError)
		return
	}
	extractDaily, err := appdb.QueryDailyExtractionSeries(ctx, a.db, 7)
	if err != nil {
		http.Error(w, "could not load extraction series: "+err.Error(), http.StatusInternalServerError)
		return
	}
	extractHourly, err := appdb.QueryHourlyExtractionSeries(ctx, a.db, 24)
	if err != nil {
		http.Error(w, "could not load extraction hourly: "+err.Error(), http.StatusInternalServerError)
		return
	}

	a.render(w, r, adminPage{
		Kind:              "admin",
		Title:             "Admin",
		Counts:            counts,
		ExtractionSum:     extractSummary,
		ProviderStats:     providerStats,
		StorageBytes:      dirSize(a.uploadsDir),
		StorageDir:        a.uploadsDir,
		ItemDailyJSON:     mustJSON(itemDaily),
		ItemHourlyJSON:    mustJSON(itemHourly),
		ExtractDailyJSON:  mustJSON(extractDaily),
		ExtractHourlyJSON: mustJSON(extractHourly),
	})
}

func (a app) adminUsers(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	users, err := appdb.QueryAdminUsers(r.Context(), a.db)
	if err != nil {
		http.Error(w, "could not load users: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.render(w, r, adminUsersPage{
		Kind:         "admin-users",
		Title:        "Admin — Users",
		Users:        users,
		NewLoginURL:  r.URL.Query().Get("login_url"),
		NewLoginName: r.URL.Query().Get("for"),
	})
}

func (a app) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	email := strings.TrimSpace(r.Form.Get("email"))
	name := strings.TrimSpace(r.Form.Get("name"))
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	user, err := a.queries.CreateUser(r.Context(), dbgen.CreateUserParams{Email: email, Name: name})
	if err != nil {
		http.Error(w, "could not create user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	loginURL, err := a.makeLoginToken(r, user.ID)
	if err != nil {
		http.Error(w, "user created but could not generate login link: "+err.Error(), http.StatusInternalServerError)
		return
	}

	displayName := user.Email
	if user.Name != "" {
		displayName = user.Name + " (" + user.Email + ")"
	}
	http.Redirect(w, r, "/admin/users?login_url="+loginURL+"&for="+displayName, http.StatusSeeOther)
}

func (a app) adminCreateLoginLink(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	userID := r.PathValue("userID")
	user, err := a.queries.GetUser(r.Context(), userID)
	if err != nil {
		a.renderNotFound(w, r)
		return
	}

	loginURL, err := a.makeLoginToken(r, user.ID)
	if err != nil {
		http.Error(w, "could not generate login link: "+err.Error(), http.StatusInternalServerError)
		return
	}

	displayName := user.Email
	if user.Name != "" {
		displayName = user.Name + " (" + user.Email + ")"
	}
	http.Redirect(w, r, "/admin/users?login_url="+loginURL+"&for="+displayName, http.StatusSeeOther)
}

func (a app) makeLoginToken(r *http.Request, userID string) (string, error) {
	token, err := share.NewToken()
	if err != nil {
		return "", err
	}
	if _, err := a.queries.CreateLoginToken(r.Context(), dbgen.CreateLoginTokenParams{UserID: userID, TokenHash: share.HashToken(token)}); err != nil {
		return "", err
	}
	return requestBaseURL(r) + "/auth/token?t=" + token, nil
}

func (a app) adminAPITokens(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	tokens, err := a.queries.ListAPITokens(r.Context())
	if err != nil {
		http.Error(w, "could not load tokens: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if tokens == nil {
		tokens = []dbgen.APIToken{}
	}
	a.render(w, r, adminAPITokensPage{
		Kind:     "admin-api-tokens",
		Title:    "Admin — API Tokens",
		Tokens:   tokens,
		NewToken: r.URL.Query().Get("new_token"),
		NewLabel: r.URL.Query().Get("for"),
	})
}

func (a app) adminCreateAPIToken(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	label := strings.TrimSpace(r.Form.Get("label"))

	user, ok := a.requireUser(w, r)
	if !ok {
		return
	}

	raw, err := share.NewToken()
	if err != nil {
		http.Error(w, "could not generate token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := a.queries.CreateAPIToken(r.Context(), dbgen.CreateAPITokenParams{UserID: user.ID, Label: label, TokenHash: share.HashToken(raw)}); err != nil {
		http.Error(w, "could not create token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/api-tokens?new_token="+raw+"&for="+label, http.StatusSeeOther)
}

func (a app) adminRevokeAPIToken(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	if err := a.queries.DeleteAPIToken(r.Context(), r.PathValue("tokenID")); err != nil {
		http.Error(w, "could not revoke token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/admin/api-tokens", http.StatusSeeOther)
}

func (a app) adminExtractions(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	const perPage = 50
	totalLogs, err := appdb.CountExtractionLogs(r.Context(), a.db)
	if err != nil {
		http.Error(w, "could not count extractions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	totalPages := (totalLogs + perPage - 1) / perPage
	if totalPages < 1 {
		totalPages = 1
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	if page > totalPages {
		page = totalPages
	}
	offset := (page - 1) * perPage
	logs, err := appdb.QueryExtractionLogsPage(r.Context(), a.db, perPage, offset)
	if err != nil {
		http.Error(w, "could not load extractions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	start, end := 0, 0
	if totalLogs > 0 && len(logs) > 0 {
		start = offset + 1
		end = offset + len(logs)
	}
	var prevURL, nextURL string
	if page > 1 {
		prevURL = adminExtractionsPageURL(r, page-1)
	}
	if page < totalPages {
		nextURL = adminExtractionsPageURL(r, page+1)
	}
	a.render(w, r, adminExtractionsPage{
		Kind:           "admin-extractions",
		Title:          "Admin — Extractions",
		RecentLogs:     logs,
		Page:           page,
		PerPage:        perPage,
		TotalLogs:      totalLogs,
		TotalPages:     totalPages,
		PrevPageURL:    prevURL,
		NextPageURL:    nextURL,
		PageStartIndex: start,
		PageEndIndex:   end,
	})
}

func parsePositiveInt(raw string, fallback int) int {
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return fallback
	}
	return n
}

func adminExtractionsPageURL(r *http.Request, page int) string {
	q := r.URL.Query()
	q.Set("page", strconv.Itoa(page))
	return r.URL.Path + "?" + q.Encode()
}

func (a app) adminExtractionDetail(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	requestID := r.PathValue("requestID")
	log, err := appdb.QueryExtractionLogByRequestID(r.Context(), a.db, requestID)
	if errors.Is(err, sql.ErrNoRows) {
		a.renderNotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "could not load extraction: "+err.Error(), http.StatusInternalServerError)
		return
	}
	a.render(w, r, adminExtractionDetailPage{
		Kind:  "admin-extraction-detail",
		Title: "Extraction " + requestID,
		Log:   log,
	})
}

func (a app) adminExtractionLogsJSON(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = n
		}
	}
	logs, err := appdb.QueryRecentExtractionLogs(r.Context(), a.db, limit)
	if err != nil {
		http.Error(w, `{"error":"could not load logs"}`, http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []appdb.ExtractionLogRow{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"logs": logs})
}

func (a app) adminExtractionLogDetailJSON(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	requestID := r.PathValue("requestID")
	log, err := appdb.QueryExtractionLogByRequestID(r.Context(), a.db, requestID)
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, `{"error":"could not load log"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(log)
}

func mustJSON(v any) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}

// dirSize returns the total size in bytes of all files under dir.
func dirSize(dir string) int64 {
	var total int64
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

// FormatBytes formats a byte count as a human-readable string (used in templates).
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
