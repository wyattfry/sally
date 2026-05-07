package generated

import (
	"context"
	"database/sql"
)

type InsertExtractionLogParams struct {
	RequestID         string
	ScheduleID        string // empty = NULL
	Provider          string
	Model             string
	PromptVersion     string
	DurationMS        int
	Success           bool
	ErrorMessage      string
	PageURL           string
	PromptTokens      int
	CompletionTokens  int
	MissingFieldCount int
	PromptText        string
	ResponseText      string
}

const insertExtractionLog = `
insert into extraction_logs
    (request_id, schedule_id, provider, model, prompt_version, duration_ms, success,
     error_message, page_url, prompt_tokens, completion_tokens, missing_fields_count,
     prompt_text, response_text)
values
    ($1, nullif($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
`

func (q *Queries) InsertExtractionLog(ctx context.Context, p InsertExtractionLogParams) error {
	_, err := q.db.ExecContext(ctx, insertExtractionLog,
		p.RequestID, p.ScheduleID, p.Provider, p.Model, p.PromptVersion,
		p.DurationMS, p.Success, p.ErrorMessage, p.PageURL,
		p.PromptTokens, p.CompletionTokens, p.MissingFieldCount,
		p.PromptText, p.ResponseText,
	)
	return err
}

// --- Admin stats types (raw SQL, called with *sql.DB directly) ---

type ExtractionSummary struct {
	Total            int64
	Successes        int64
	Last24h          int64
	Last7d           int64
	AvgDurMS         float64
	TotalPromptTok   int64
	TotalCompleteTok int64
}

type ExtractionProviderStat struct {
	Provider string
	Model    string
	Total    int64
	Successes int64
	AvgDurMS float64
}

type ExtractionLogRow struct {
	RequestID         string
	Provider          string
	Model             string
	PromptVersion     string
	DurationMS        int
	Success           bool
	Error             string
	PageURL           string
	ScheduleID        string
	CreatedAt         string
	PromptTokens      int
	CompletionTokens  int
	MissingFieldCount int
	PromptText        string
	ResponseText      string
}

type DailyPoint struct {
	Date  string
	Count int64
	Extra int64 // second series (e.g. errors for extraction chart)
}

type AdminUserRow struct {
	ID            string
	Email         string
	Name          string
	CreatedAt     string
	ProjectCount  int64
	ScheduleCount int64
	ItemCount     int64
	LastItemAt    string // empty if no items
}

func QueryExtractionSummary(ctx context.Context, db *sql.DB) (ExtractionSummary, error) {
	row := db.QueryRowContext(ctx, `
		select
			count(*)                                                          as total,
			count(*) filter (where success)                                   as successes,
			count(*) filter (where created_at > now() - interval '24 hours') as last_24h,
			count(*) filter (where created_at > now() - interval '7 days')   as last_7d,
			coalesce(avg(duration_ms) filter (where success), 0)             as avg_dur_ms,
			coalesce(sum(prompt_tokens), 0)                                  as total_prompt_tok,
			coalesce(sum(completion_tokens), 0)                              as total_complete_tok
		from extraction_logs
	`)
	var s ExtractionSummary
	err := row.Scan(&s.Total, &s.Successes, &s.Last24h, &s.Last7d, &s.AvgDurMS,
		&s.TotalPromptTok, &s.TotalCompleteTok)
	return s, err
}

func QueryExtractionProviderStats(ctx context.Context, db *sql.DB) ([]ExtractionProviderStat, error) {
	rows, err := db.QueryContext(ctx, `
		select
			provider, model,
			count(*)                                              as total,
			count(*) filter (where success)                       as successes,
			coalesce(avg(duration_ms) filter (where success), 0) as avg_dur_ms
		from extraction_logs
		group by provider, model
		order by total desc
		limit 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExtractionProviderStat
	for rows.Next() {
		var s ExtractionProviderStat
		if err := rows.Scan(&s.Provider, &s.Model, &s.Total, &s.Successes, &s.AvgDurMS); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func QueryRecentExtractionLogs(ctx context.Context, db *sql.DB, limit int) ([]ExtractionLogRow, error) {
	rows, err := db.QueryContext(ctx, `
		select
			request_id, provider, model, prompt_version, duration_ms, success,
			error_message, page_url,
			coalesce(schedule_id::text, '') as schedule_id,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as created_at,
			prompt_tokens, completion_tokens, missing_fields_count
		from extraction_logs
		order by created_at desc
		limit $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ExtractionLogRow
	for rows.Next() {
		var r ExtractionLogRow
		if err := rows.Scan(&r.RequestID, &r.Provider, &r.Model, &r.PromptVersion, &r.DurationMS,
			&r.Success, &r.Error, &r.PageURL, &r.ScheduleID, &r.CreatedAt,
			&r.PromptTokens, &r.CompletionTokens, &r.MissingFieldCount); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func QueryExtractionLogByRequestID(ctx context.Context, db *sql.DB, requestID string) (ExtractionLogRow, error) {
	row := db.QueryRowContext(ctx, `
		select
			request_id, provider, model, prompt_version, duration_ms, success,
			error_message, page_url,
			coalesce(schedule_id::text, '') as schedule_id,
			to_char(created_at at time zone 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"') as created_at,
			prompt_tokens, completion_tokens, missing_fields_count,
			prompt_text, response_text
		from extraction_logs
		where request_id = $1
		limit 1
	`, requestID)
	var r ExtractionLogRow
	err := row.Scan(&r.RequestID, &r.Provider, &r.Model, &r.PromptVersion, &r.DurationMS,
		&r.Success, &r.Error, &r.PageURL, &r.ScheduleID, &r.CreatedAt,
		&r.PromptTokens, &r.CompletionTokens, &r.MissingFieldCount,
		&r.PromptText, &r.ResponseText)
	return r, err
}

type AdminTableCounts struct {
	Users         int64
	Projects      int64
	Schedules     int64
	Items         int64
	NewUsers7d    int64
	NewProjects7d int64
	NewItems7d    int64
}

func QueryAdminTableCounts(ctx context.Context, db *sql.DB) (AdminTableCounts, error) {
	row := db.QueryRowContext(ctx, `
		select
			(select count(*) from users),
			(select count(*) from projects),
			(select count(*) from schedules),
			(select count(*) from schedule_items),
			(select count(*) from users           where created_at > now() - interval '7 days'),
			(select count(*) from projects        where created_at > now() - interval '7 days'),
			(select count(*) from schedule_items  where created_at > now() - interval '7 days')
	`)
	var c AdminTableCounts
	err := row.Scan(&c.Users, &c.Projects, &c.Schedules, &c.Items,
		&c.NewUsers7d, &c.NewProjects7d, &c.NewItems7d)
	return c, err
}

func QueryDailyItemSeries(ctx context.Context, db *sql.DB, days int) ([]DailyPoint, error) {
	rows, err := db.QueryContext(ctx, `
		select
			to_char(day, 'YYYY-MM-DD') as date,
			count(si.id)               as items
		from generate_series(
			current_date - ($1 - 1) * interval '1 day',
			current_date,
			interval '1 day'
		) as day
		left join schedule_items si on si.created_at::date = day::date
		group by day
		order by day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyPoint
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func QueryDailyExtractionSeries(ctx context.Context, db *sql.DB, days int) ([]DailyPoint, error) {
	rows, err := db.QueryContext(ctx, `
		select
			to_char(day, 'YYYY-MM-DD')                      as date,
			count(el.id)                                     as total,
			count(el.id) filter (where not el.success)       as errors
		from generate_series(
			current_date - ($1 - 1) * interval '1 day',
			current_date,
			interval '1 day'
		) as day
		left join extraction_logs el on el.created_at::date = day::date
		group by day
		order by day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyPoint
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Count, &p.Extra); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func QueryAdminUsers(ctx context.Context, db *sql.DB) ([]AdminUserRow, error) {
	rows, err := db.QueryContext(ctx, `
		select
			u.id::text,
			u.email,
			coalesce(u.name, '') as name,
			to_char(u.created_at at time zone 'UTC', 'YYYY-MM-DD') as created_at,
			count(distinct p.id)   as project_count,
			count(distinct s.id)   as schedule_count,
			count(distinct si.id)  as item_count,
			coalesce(to_char(max(si.created_at) at time zone 'UTC', 'YYYY-MM-DD'), '') as last_item_at
		from users u
		left join projects p  on p.owner_user_id = u.id
		left join schedules s on s.project_id = p.id
		left join schedule_items si on si.schedule_id = s.id
		group by u.id, u.email, u.name, u.created_at
		order by item_count desc, u.created_at desc
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdminUserRow
	for rows.Next() {
		var r AdminUserRow
		if err := rows.Scan(&r.ID, &r.Email, &r.Name, &r.CreatedAt,
			&r.ProjectCount, &r.ScheduleCount, &r.ItemCount, &r.LastItemAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// QueryHourlyItemSeries returns hour-bucket item counts for the last `hours` hours.
// Used for 1h / 3h / 24h chart ranges.
func QueryHourlyItemSeries(ctx context.Context, db *sql.DB, hours int) ([]DailyPoint, error) {
	rows, err := db.QueryContext(ctx, `
		select
			to_char(bucket, 'MM-DD HH24:00') as date,
			count(si.id)                      as count
		from generate_series(
			date_trunc('hour', now()) - ($1 - 1) * interval '1 hour',
			date_trunc('hour', now()),
			interval '1 hour'
		) as bucket
		left join schedule_items si
			on  si.created_at >= bucket
			and si.created_at <  bucket + interval '1 hour'
		group by bucket
		order by bucket
	`, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyPoint
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Count); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// QueryHourlyExtractionSeries returns hour-bucket extraction counts for the last `hours` hours.
func QueryHourlyExtractionSeries(ctx context.Context, db *sql.DB, hours int) ([]DailyPoint, error) {
	rows, err := db.QueryContext(ctx, `
		select
			to_char(bucket, 'MM-DD HH24:00') as date,
			count(el.id)                                   as total,
			count(el.id) filter (where not el.success)     as errors
		from generate_series(
			date_trunc('hour', now()) - ($1 - 1) * interval '1 hour',
			date_trunc('hour', now()),
			interval '1 hour'
		) as bucket
		left join extraction_logs el
			on  el.created_at >= bucket
			and el.created_at <  bucket + interval '1 hour'
		group by bucket
		order by bucket
	`, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DailyPoint
	for rows.Next() {
		var p DailyPoint
		if err := rows.Scan(&p.Date, &p.Count, &p.Extra); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
