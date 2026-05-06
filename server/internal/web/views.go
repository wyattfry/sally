package web

import queries "sally/server/internal/db/generated"

type projectListItem struct {
	Project     queries.Project
	ThumbImages []string
}

type projectsPage struct {
	Kind     string
	Title    string
	Projects []projectListItem
}

type projectDetailPage struct {
	Kind           string
	Title          string
	Project        queries.Project
	Schedules      []scheduleWithItems
	FirstItemImage string
	ActiveLink     *queries.ProjectShareLink
	ShareBaseURL   string
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
	Kind                 string
	Title                string
	Counts               queries.AdminTableCounts
	ExtractionSum        queries.ExtractionSummary
	ProviderStats        []queries.ExtractionProviderStat
	StorageBytes         int64
	StorageDir           string
	ItemDailyJSON        string // daily series, 7d
	ItemHourlyJSON       string // hourly series, 24h
	ExtractDailyJSON     string // daily series, 7d
	ExtractHourlyJSON    string // hourly series, 24h
}

type adminUsersPage struct {
	Kind  string
	Title string
	Users []queries.AdminUserRow
}

type adminExtractionsPage struct {
	Kind       string
	Title      string
	RecentLogs []queries.ExtractionLogRow
}
