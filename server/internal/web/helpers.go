package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/presets"
)

// naturalLess compares two strings using a natural-sort order: runs of digits
// are compared numerically, runs of non-digits lexicographically (case-insensitive).
// So "E-2" < "E-10" and "PT-1A" < "PT-1B" < "PT-2".
func naturalLess(a, b string) bool {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		ad := a[i] >= '0' && a[i] <= '9'
		bd := b[j] >= '0' && b[j] <= '9'
		if ad && bd {
			ai := i
			for i < len(a) && a[i] >= '0' && a[i] <= '9' {
				i++
			}
			bj := j
			for j < len(b) && b[j] >= '0' && b[j] <= '9' {
				j++
			}
			an, _ := strconv.Atoi(a[ai:i])
			bn, _ := strconv.Atoi(b[bj:j])
			if an != bn {
				return an < bn
			}
		} else {
			if a[i] != b[j] {
				return a[i] < b[j]
			}
			i++
			j++
		}
	}
	return len(a)-i < len(b)-j
}

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
			if token := r.Header.Get("X-Session-Token"); token != "" {
				email, ok = ValidateSessionToken(a.sessionSecret, token)
			}
		}
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
		if col.Key == "room" {
			continue // room has its own DB column; not stored in data JSONB
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
	dm["room"] = item.Room
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

type roomGroup struct {
	Room  string
	Items []scheduleItemView
}

type scheduleWithItems struct {
	Schedule queries.Schedule
	Columns  []queries.ScheduleColumn
	Groups   []roomGroup
	// Populated by computeContractorTotals when rendering in contractor view.
	ContractorTotals *contractorTotals
}

func groupByRoom(items []scheduleItemView) []roomGroup {
	seen := map[string]int{}
	var groups []roomGroup
	for _, item := range items {
		if idx, ok := seen[item.Room]; ok {
			groups[idx].Items = append(groups[idx].Items, item)
		} else {
			seen[item.Room] = len(groups)
			groups = append(groups, roomGroup{Room: item.Room, Items: []scheduleItemView{item}})
		}
	}
	return groups
}

func (a app) scheduleSummaries(ctx context.Context, projectID string) ([]scheduleSummary, error) {
	schedules, err := a.queries.ListSchedulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]scheduleSummary, 0, len(schedules))
	for _, s := range schedules {
		items, err := a.queries.ListScheduleItems(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		last := s.UpdatedAt
		for _, it := range items {
			if it.UpdatedAt.After(last) {
				last = it.UpdatedAt
			}
		}
		out = append(out, scheduleSummary{
			Schedule:      s,
			ItemCount:     len(items),
			LastUpdated:   last,
			PreviewImages: collectPreviewImages(items, 3),
		})
	}
	return out, nil
}

// scheduleWithItemsByID loads a single schedule (with columns, items, room grouping)
// matching the shape used by schedulesWithItems.
func (a app) scheduleWithItemsByID(ctx context.Context, schedule queries.Schedule) (scheduleWithItems, error) {
	cols, err := a.queries.ListScheduleColumns(ctx, schedule.ID)
	if err != nil {
		return scheduleWithItems{}, err
	}
	rawItems, err := a.queries.ListScheduleItems(ctx, schedule.ID)
	if err != nil {
		return scheduleWithItems{}, err
	}
	views := make([]scheduleItemView, len(rawItems))
	for i, it := range rawItems {
		views[i] = toItemView(it)
	}
	for _, col := range cols {
		if col.Key == "code" {
			sort.Slice(views, func(i, j int) bool {
				return naturalLess(views[i].DataMap["code"], views[j].DataMap["code"])
			})
			break
		}
	}
	return scheduleWithItems{
		Schedule: schedule,
		Columns:  cols,
		Groups:   groupByRoom(views),
	}, nil
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
					return naturalLess(views[i].DataMap["code"], views[j].DataMap["code"])
				})
				break
			}
		}
		result = append(result, scheduleWithItems{
			Schedule: schedule,
			Columns:  cols,
			Groups:   groupByRoom(views),
		})
	}
	return result, nil
}

// scheduleSummariesWithContractorTotals enriches each summary with a
// per-schedule contractor subtotal block. Used by the contractor share
// view's project page; the architect path uses scheduleSummaries.
func (a app) scheduleSummariesWithContractorTotals(ctx context.Context, projectID string, _ int, staleRedDays int) ([]scheduleSummary, error) {
	schedules, err := a.queries.ListSchedulesByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	out := make([]scheduleSummary, 0, len(schedules))
	for _, s := range schedules {
		cols, _ := a.queries.ListScheduleColumns(ctx, s.ID)
		rawItems, _ := a.queries.ListScheduleItems(ctx, s.ID)
		views := make([]scheduleItemView, len(rawItems))
		for i, it := range rawItems {
			views[i] = toItemView(it)
		}
		last := s.UpdatedAt
		for _, it := range rawItems {
			if it.UpdatedAt.After(last) {
				last = it.UpdatedAt
			}
		}
		totals := computeContractorTotals(scheduleWithItems{
			Schedule: s,
			Columns:  cols,
			Groups:   groupByRoom(views),
		}, staleRedDays)
		out = append(out, scheduleSummary{
			Schedule:         s,
			ItemCount:        len(rawItems),
			LastUpdated:      last,
			PreviewImages:    collectPreviewImages(rawItems, 3),
			ContractorTotals: totals,
		})
	}
	return out, nil
}

// collectPreviewImages returns up to n non-empty SourceImageUrl values from
// a slice of items, for use as thumbnail previews on the schedule list row.
func collectPreviewImages(items []queries.ScheduleItem, n int) []string {
	var out []string
	for _, it := range items {
		if it.SourceImageUrl != "" {
			out = append(out, it.SourceImageUrl)
			if len(out) >= n {
				break
			}
		}
	}
	return out
}

// priceParseRE matches a single dollar amount like "$135.38", "$1,519.20",
// "$199", optionally preceded by "Was " or similar. Anything ambiguous (a
// range "$X - $Y", "starting at", multi-currency) deliberately does NOT
// match — those items are excluded from the subtotal and flagged in the
// totals warning so the contractor sees what wasn't counted.
var priceParseRE = regexp.MustCompile(`^[\s$]*\$?(\d{1,3}(?:,\d{3})*(?:\.\d{1,2})?)\s*$`)
var priceRangeRE = regexp.MustCompile(`[-–—]|\bto\b|starting`)

// computeContractorTotals walks a schedule and produces the price aggregate
// plus the named exclusions. Never excludes silently — every reason a row
// doesn't contribute to the subtotal lands in one of MissingPrice /
// RangePrice / StalePrice so the contractor can chase it down.
func computeContractorTotals(sw scheduleWithItems, staleRedDays int) *contractorTotals {
	t := &contractorTotals{}
	staleCutoff := time.Time{}
	if staleRedDays > 0 {
		staleCutoff = time.Now().AddDate(0, 0, -staleRedDays)
	}
	for _, g := range sw.Groups {
		for _, it := range g.Items {
			t.TotalItems++
			code := strings.TrimSpace(it.DataMap["code"])
			if code == "" {
				code = "—"
			}
			raw := strings.TrimSpace(it.DataMap["price"])
			if raw == "" {
				t.MissingPrice = append(t.MissingPrice, code)
				continue
			}
			if priceRangeRE.MatchString(strings.ToLower(raw)) {
				t.RangePrice = append(t.RangePrice, code)
				continue
			}
			cents, ok := parsePriceCents(raw)
			if !ok {
				t.MissingPrice = append(t.MissingPrice, code)
				continue
			}
			// Stale snapshot: still counts toward the subtotal (we have a
			// price), but flagged so the contractor knows it might be wrong.
			if pricedAt, err := time.Parse(time.RFC3339, it.DataMap["priced_at"]); err == nil {
				if !staleCutoff.IsZero() && pricedAt.Before(staleCutoff) {
					t.StalePrice = append(t.StalePrice, code)
				}
			}
			t.SubtotalCents += cents
			t.PricedCount++
		}
	}
	t.SubtotalDisplay = formatCents(t.SubtotalCents)
	return t
}

func parsePriceCents(raw string) (int64, bool) {
	m := priceParseRE.FindStringSubmatch(raw)
	if m == nil {
		return 0, false
	}
	cleaned := strings.ReplaceAll(m[1], ",", "")
	parts := strings.SplitN(cleaned, ".", 2)
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, false
	}
	var cents int64
	if len(parts) == 2 {
		// Pad to 2 digits (handles "$3.5" → 350 cents).
		p := parts[1]
		if len(p) == 1 {
			p += "0"
		} else if len(p) > 2 {
			p = p[:2]
		}
		c, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return 0, false
		}
		cents = c
	}
	return dollars*100 + cents, true
}

func formatCents(c int64) string {
	dollars := c / 100
	cents := c % 100
	// Thousand separators.
	in := strconv.FormatInt(dollars, 10)
	var b strings.Builder
	for i, r := range in {
		if i > 0 && (len(in)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return fmt.Sprintf("$%s.%02d", b.String(), cents)
}
