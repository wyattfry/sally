package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"testing"
	"time"

	queries "sally/server/internal/db/generated"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestProjectScheduleItemQueries(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	q := queries.New(conn)
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "architect-query-test@example.com",
		Name:  "Query Test Architect",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Query Test Project",
		Address:     "24 School St.",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	schedule, err := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Bath",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	itemData, _ := json.Marshal(map[string]string{
		"code":               "B-01",
		"title":              "Wall Faucet",
		"description":        "Wall-mounted faucet with rough valve.",
		"manufacturer":       "Example Co.",
		"model_number":       "WF-200",
		"finish":             "Polished Chrome",
		"finish_model_number": "WF-200-PC",
		"notes":              "Verify rough-in.",
	})
	item, err := q.CreateScheduleItem(context.Background(), queries.CreateScheduleItemParams{
		ScheduleID:     schedule.ID,
		Data:           itemData,
		SourceUrl:      "https://example.com/products/wf-200",
		SourceTitle:    "Example Co. WF-200 Wall Faucet",
		SourceImageUrl: "https://example.com/faucet.jpg",
		SourcePdfLinks: []string{"https://example.com/spec-sheet.pdf"},
		Position:       1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	items, err := q.ListScheduleItems(context.Background(), schedule.ID)
	if err != nil {
		t.Fatalf("list items: %v", err)
	}
	if len(items) != 1 || items[0].ID != item.ID {
		t.Fatalf("expected created item in list, got %#v", items)
	}
}

func TestProjectUpdatedAtBumpsOnChildChanges(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	q := queries.New(conn)
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "trigger-test@example.com",
		Name:  "Trigger Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Trigger Test Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	t0 := project.UpdatedAt

	// Creating a schedule must bump project.updated_at.
	time.Sleep(time.Millisecond)
	schedule, err := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Bath",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	after1, err := q.GetProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !after1.UpdatedAt.After(t0) {
		t.Fatalf("expected updated_at to advance after schedule create: was %v, now %v", t0, after1.UpdatedAt)
	}
	t1 := after1.UpdatedAt

	// Adding an item must bump project.updated_at.
	time.Sleep(time.Millisecond)
	item, err := q.CreateScheduleItem(context.Background(), queries.CreateScheduleItemParams{
		ScheduleID:     schedule.ID,
		Data:           json.RawMessage(`{"title":"Wall Faucet"}`),
		SourcePdfLinks: []string{},
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}
	after2, err := q.GetProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !after2.UpdatedAt.After(t1) {
		t.Fatalf("expected updated_at to advance after item create: was %v, now %v", t1, after2.UpdatedAt)
	}
	t2 := after2.UpdatedAt

	// Deleting an item must also bump project.updated_at.
	time.Sleep(time.Millisecond)
	if err := q.DeleteScheduleItem(context.Background(), item.ID); err != nil {
		t.Fatalf("delete item: %v", err)
	}
	after3, err := q.GetProject(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !after3.UpdatedAt.After(t2) {
		t.Fatalf("expected updated_at to advance after item delete: was %v, now %v", t2, after3.UpdatedAt)
	}
}

// TestAdminQueries exercises every hand-written admin SQL function against the
// real schema. Any column-name typo (like owner_id vs owner_user_id) will
// surface here rather than at runtime.
func TestAdminQueries(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	ctx := context.Background()
	q := queries.New(conn)

	// Seed minimal data so the join-heavy queries have something to traverse.
	user, err := q.CreateUser(ctx, queries.CreateUserParams{
		Email: "admin-query-test@example.com",
		Name:  "Admin Query Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(ctx, queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Admin Test Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	schedule, err := q.CreateSchedule(ctx, queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Admin Test Schedule",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	_, err = q.CreateScheduleItem(ctx, queries.CreateScheduleItemParams{
		ScheduleID:     schedule.ID,
		Data:           json.RawMessage(`{"title":"Test Item"}`),
		SourcePdfLinks: []string{},
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	if err := q.InsertExtractionLog(ctx, queries.InsertExtractionLogParams{
		RequestID:        "test-req-1",
		ScheduleID:       schedule.ID,
		Provider:         "openai",
		Model:            "gpt-4o",
		DurationMs:       1234,
		Success:          true,
		PromptTokens:     500,
		CompletionTokens: 120,
	}); err != nil {
		t.Fatalf("insert extraction log: %v", err)
	}

	t.Run("QueryAdminTableCounts", func(t *testing.T) {
		if _, err := QueryAdminTableCounts(ctx, conn); err != nil {
			t.Fatalf("QueryAdminTableCounts: %v", err)
		}
	})

	t.Run("QueryExtractionSummary", func(t *testing.T) {
		if _, err := QueryExtractionSummary(ctx, conn); err != nil {
			t.Fatalf("QueryExtractionSummary: %v", err)
		}
	})

	t.Run("QueryExtractionProviderStats", func(t *testing.T) {
		if _, err := QueryExtractionProviderStats(ctx, conn); err != nil {
			t.Fatalf("QueryExtractionProviderStats: %v", err)
		}
	})

	t.Run("QueryRecentExtractionLogs", func(t *testing.T) {
		if _, err := QueryRecentExtractionLogs(ctx, conn, 10); err != nil {
			t.Fatalf("QueryRecentExtractionLogs: %v", err)
		}
	})

	t.Run("QueryAdminUsers", func(t *testing.T) {
		rows, err := QueryAdminUsers(ctx, conn)
		if err != nil {
			t.Fatalf("QueryAdminUsers: %v", err)
		}
		var found bool
		for _, r := range rows {
			if r.Email == user.Email {
				found = true
				if r.ProjectCount < 1 {
					t.Errorf("expected at least 1 project for test user, got %d", r.ProjectCount)
				}
			}
		}
		if !found {
			t.Errorf("test user not found in QueryAdminUsers result")
		}
	})

	t.Run("QueryDailyItemSeries", func(t *testing.T) {
		pts, err := QueryDailyItemSeries(ctx, conn, 28)
		if err != nil {
			t.Fatalf("QueryDailyItemSeries: %v", err)
		}
		if len(pts) != 28 {
			t.Errorf("expected 28 points, got %d", len(pts))
		}
	})

	t.Run("QueryDailyExtractionSeries", func(t *testing.T) {
		pts, err := QueryDailyExtractionSeries(ctx, conn, 28)
		if err != nil {
			t.Fatalf("QueryDailyExtractionSeries: %v", err)
		}
		if len(pts) != 28 {
			t.Errorf("expected 28 points, got %d", len(pts))
		}
	})

	t.Run("QueryHourlyItemSeries", func(t *testing.T) {
		pts, err := QueryHourlyItemSeries(ctx, conn, 24)
		if err != nil {
			t.Fatalf("QueryHourlyItemSeries: %v", err)
		}
		if len(pts) != 24 {
			t.Errorf("expected 24 points, got %d", len(pts))
		}
	})

	t.Run("QueryHourlyExtractionSeries", func(t *testing.T) {
		pts, err := QueryHourlyExtractionSeries(ctx, conn, 24)
		if err != nil {
			t.Fatalf("QueryHourlyExtractionSeries: %v", err)
		}
		if len(pts) != 24 {
			t.Errorf("expected 24 points, got %d", len(pts))
		}
	})
}

func TestProjectMemberQueries(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	q := queries.New(conn)
	ctx := context.Background()

	owner, err := q.CreateUser(ctx, queries.CreateUserParams{
		Email: "pm-query-owner@example.com",
		Name:  "PM Query Owner",
	})
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	invitee, err := q.CreateUser(ctx, queries.CreateUserParams{
		Email: "pm-query-invitee@example.com",
		Name:  "PM Query Invitee",
	})
	if err != nil {
		t.Fatalf("create invitee: %v", err)
	}

	project, err := q.CreateProject(ctx, queries.CreateProjectParams{
		OwnerUserID: owner.ID,
		Name:        "PM Query Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	t.Run("AddProjectMember", func(t *testing.T) {
		err := q.AddProjectMember(ctx, queries.AddProjectMemberParams{
			ProjectID:       project.ID,
			UserID:          invitee.ID,
			InvitedByUserID: owner.ID,
		})
		if err != nil {
			t.Fatalf("AddProjectMember: %v", err)
		}
	})

	t.Run("AddProjectMember idempotent", func(t *testing.T) {
		// Second add should not error (on conflict do nothing).
		err := q.AddProjectMember(ctx, queries.AddProjectMemberParams{
			ProjectID:       project.ID,
			UserID:          invitee.ID,
			InvitedByUserID: owner.ID,
		})
		if err != nil {
			t.Fatalf("AddProjectMember duplicate: %v", err)
		}
	})

	t.Run("GetProjectMember", func(t *testing.T) {
		m, err := q.GetProjectMember(ctx, queries.GetProjectMemberParams{
			ProjectID: project.ID,
			UserID:    invitee.ID,
		})
		if err != nil {
			t.Fatalf("GetProjectMember: %v", err)
		}
		if m.UserID != invitee.ID {
			t.Errorf("expected user_id %s, got %s", invitee.ID, m.UserID)
		}
		if m.InvitedByUserID != owner.ID {
			t.Errorf("expected invited_by %s, got %s", owner.ID, m.InvitedByUserID)
		}
	})

	t.Run("ListProjectMembersWithUser", func(t *testing.T) {
		members, err := q.ListProjectMembersWithUser(ctx, project.ID)
		if err != nil {
			t.Fatalf("ListProjectMembersWithUser: %v", err)
		}
		if len(members) == 0 {
			t.Fatal("expected at least one member")
		}
		found := false
		for _, m := range members {
			if m.UserID == invitee.ID {
				found = true
				if m.UserEmail != invitee.Email {
					t.Errorf("expected email %s, got %s", invitee.Email, m.UserEmail)
				}
				if m.UserName != invitee.Name {
					t.Errorf("expected name %s, got %s", invitee.Name, m.UserName)
				}
			}
		}
		if !found {
			t.Errorf("invitee not found in member list")
		}
	})

	t.Run("ListSharedProjects", func(t *testing.T) {
		shared, err := q.ListSharedProjects(ctx, invitee.ID)
		if err != nil {
			t.Fatalf("ListSharedProjects: %v", err)
		}
		found := false
		for _, p := range shared {
			if p.ID == project.ID {
				found = true
			}
		}
		if !found {
			t.Errorf("project %s not found in invitee's shared projects", project.ID)
		}
	})

	t.Run("RemoveProjectMember", func(t *testing.T) {
		err := q.RemoveProjectMember(ctx, queries.RemoveProjectMemberParams{
			ProjectID: project.ID,
			UserID:    invitee.ID,
		})
		if err != nil {
			t.Fatalf("RemoveProjectMember: %v", err)
		}
		members, err := q.ListProjectMembersWithUser(ctx, project.ID)
		if err != nil {
			t.Fatalf("list after remove: %v", err)
		}
		for _, m := range members {
			if m.UserID == invitee.ID {
				t.Errorf("invitee still present after removal")
			}
		}
	})
}
