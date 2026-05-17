package web

import (
	"html/template"
	"time"

	queries "sally/server/internal/db/generated"
)

type projectListItem struct {
	Project          queries.Project
	ThumbImages      []string
	OwnerDisplayName string
}

type projectsPage struct {
	Kind           string
	Title          string
	Projects       []projectListItem
	SharedProjects []projectListItem
}

type projectDetailPage struct {
	Kind             string
	Title            string
	Project          queries.Project
	Schedules        []scheduleSummary
	FirstItemImage   string
	ActiveLink       *queries.ProjectShareLink
	ShareBaseURL     string
	Members          []queries.ProjectMemberWithUser
	IsOwner          bool
	MemberError      string
	OwnerDisplayName string
}

type scheduleDetailPage struct {
	Kind             string
	Title            string
	Project          queries.Project
	Schedule         scheduleWithItems
	IsOwner          bool
	OwnerDisplayName string
}

type scheduleSummary struct {
	Schedule    queries.Schedule
	ItemCount   int
	LastUpdated time.Time
}

type projectEditPage struct {
	Kind    string
	Title   string
	Project queries.Project
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

type aboutPage struct {
	Kind  string
	Title string
}

type notFoundPage struct {
	Kind  string
	Title string
}

type publicSharePage struct {
	Kind      string
	Title     string
	Project   queries.Project
	Schedules []scheduleWithItems
}

type adminPage struct {
	Kind              string
	Title             string
	Counts            queries.AdminTableCounts
	ExtractionSum     queries.ExtractionSummary
	ProviderStats     []queries.ExtractionProviderStat
	StorageBytes      int64
	StorageDir        string
	ItemDailyJSON     template.JS // daily series, 7d
	ItemHourlyJSON    template.JS // hourly series, 24h
	ExtractDailyJSON  template.JS // daily series, 7d
	ExtractHourlyJSON template.JS // hourly series, 24h
}

type adminUsersPage struct {
	Kind         string
	Title        string
	Users        []queries.AdminUserRow
	NewLoginURL  string
	NewLoginName string
}

type adminAPITokensPage struct {
	Kind     string
	Title    string
	Tokens   []queries.APIToken
	NewToken string // shown once immediately after creation
	NewLabel string
}

type adminExtractionsPage struct {
	Kind       string
	Title      string
	RecentLogs []queries.ExtractionLogRow
}

type adminExtractionDetailPage struct {
	Kind  string
	Title string
	Log   queries.ExtractionLogRow
}
