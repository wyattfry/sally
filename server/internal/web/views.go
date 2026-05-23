package web

import (
	"html/template"
	"time"

	appdb "sally/server/internal/db"
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
	Members          []queries.ListProjectMembersWithUserRow
	IsOwner          bool
	MemberError      string
	OwnerDisplayName string
	// ViewMode controls per-view affordances; "architect" (default,
	// authenticated owner/editor) or "contractor" (anonymous share-link
	// visitor). Contractor mode renders price/lead/stock columns + totals.
	ViewMode string
	// ShareToken is non-empty in contractor view; used to build the
	// schedule-page links so the share-token survives the click-through.
	ShareToken string
}

type scheduleDetailPage struct {
	Kind             string
	Title            string
	Project          queries.Project
	Schedule         scheduleWithItems
	IsOwner          bool
	OwnerDisplayName string
	// Share-box context — surfaced in the project-header strip rendered at
	// the top of the schedule page (architect view).
	ActiveLink   *queries.ProjectShareLink
	ShareBaseURL string
	// FirstItemImage is unused on the schedule page but required by the
	// shared project-header template, which uses it as a hero fallback. Left
	// empty here so the partial's {{else if .FirstItemImage}} branch is
	// safely skipped.
	FirstItemImage string
	// Contractor-view affordances. See projectDetailPage for the contract.
	ViewMode       string
	ShareToken     string
	StaleAmberDays int
	StaleRedDays   int
}

type scheduleSummary struct {
	Schedule    queries.Schedule
	ItemCount   int
	LastUpdated time.Time
	// Up to 3 item thumbnail URLs shown as preview chips on the schedule
	// list row (contractor view entices clicking; architect gets context).
	PreviewImages []string
	// Populated in contractor view: per-schedule subtotal + counts of items
	// excluded for various reasons (no price, price range, stale snapshot).
	ContractorTotals *contractorTotals
}

// contractorTotals carries the price aggregation for a schedule, with the
// exclusions called out explicitly. The contractor view never sums silently —
// every reason a row didn't contribute to the subtotal is named alongside the
// number.
type contractorTotals struct {
	SubtotalCents   int64    // Σ of items whose price parses as a single dollar amount
	SubtotalDisplay string   // formatted "$4,213.00"
	TotalItems      int      // total item count in the schedule
	PricedCount     int      // items contributing to subtotal
	MissingPrice    []string // codes of items with no price
	RangePrice      []string // codes of items with price ranges (excluded from sum)
	StalePrice      []string // codes whose snapshot is past the red threshold
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

type signInPage struct {
	Kind  string
	Title string
}

type aboutPage struct {
	Kind  string
	Title string
}

type staticPage struct {
	Kind  string
	Title string
}

type notFoundPage struct {
	Kind  string
	Title string
}

type adminPage struct {
	Kind              string
	Title             string
	Counts            appdb.AdminTableCounts
	ExtractionSum     appdb.ExtractionSummary
	ProviderStats     []appdb.ExtractionProviderStat
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
	Users        []appdb.AdminUserRow
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
	RecentLogs []appdb.ExtractionLogRow
}

type adminExtractionDetailPage struct {
	Kind  string
	Title string
	Log   appdb.ExtractionLogRow
}
