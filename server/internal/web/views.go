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
