package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	dbgen "sally/server/internal/db/generated"
)

// requireAdmin checks admin access and returns false (already responded) if denied.
func (a app) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if a.oauthConfig != nil {
		user, ok := a.requireUser(w, r)
		if !ok {
			return false
		}
		if a.adminEmail == "" || user.Email != a.adminEmail {
			http.Error(w, "forbidden", http.StatusForbidden)
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

	counts, err := dbgen.QueryAdminTableCounts(ctx, a.db)
	if err != nil {
		http.Error(w, "could not load counts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	extractSummary, err := dbgen.QueryExtractionSummary(ctx, a.db)
	if err != nil {
		http.Error(w, "could not load extraction summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	providerStats, err := dbgen.QueryExtractionProviderStats(ctx, a.db)
	if err != nil {
		http.Error(w, "could not load provider stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	itemDaily, err := dbgen.QueryDailyItemSeries(ctx, a.db, 7)
	if err != nil {
		http.Error(w, "could not load item series: "+err.Error(), http.StatusInternalServerError)
		return
	}
	itemHourly, err := dbgen.QueryHourlyItemSeries(ctx, a.db, 24)
	if err != nil {
		http.Error(w, "could not load item hourly: "+err.Error(), http.StatusInternalServerError)
		return
	}
	extractDaily, err := dbgen.QueryDailyExtractionSeries(ctx, a.db, 7)
	if err != nil {
		http.Error(w, "could not load extraction series: "+err.Error(), http.StatusInternalServerError)
		return
	}
	extractHourly, err := dbgen.QueryHourlyExtractionSeries(ctx, a.db, 24)
	if err != nil {
		http.Error(w, "could not load extraction hourly: "+err.Error(), http.StatusInternalServerError)
		return
	}

	render(w, adminPage{
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
	users, err := dbgen.QueryAdminUsers(r.Context(), a.db)
	if err != nil {
		http.Error(w, "could not load users: "+err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, adminUsersPage{
		Kind:  "admin-users",
		Title: "Admin — Users",
		Users: users,
	})
}

func (a app) adminExtractions(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	logs, err := dbgen.QueryRecentExtractionLogs(r.Context(), a.db, 500)
	if err != nil {
		http.Error(w, "could not load extractions: "+err.Error(), http.StatusInternalServerError)
		return
	}
	render(w, adminExtractionsPage{
		Kind:       "admin-extractions",
		Title:      "Admin — Extractions",
		RecentLogs: logs,
	})
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
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
