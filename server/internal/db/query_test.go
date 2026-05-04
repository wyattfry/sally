package db

import (
	"context"
	"database/sql"
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

	item, err := q.CreateScheduleItem(context.Background(), queries.CreateScheduleItemParams{
		ScheduleID:        schedule.ID,
		Code:              "B-01",
		Title:             "Wall Faucet",
		Description:       "Wall-mounted faucet with rough valve.",
		Manufacturer:      "Example Co.",
		ModelNumber:       "WF-200",
		Finish:            "Polished Chrome",
		FinishModelNumber: "WF-200-PC",
		Notes:             "Verify rough-in.",
		SourceUrl:         "https://example.com/products/wf-200",
		SourceTitle:       "Example Co. WF-200 Wall Faucet",
		SourceImageUrl:    "https://example.com/faucet.jpg",
		SourcePdfLinks:    []string{"https://example.com/spec-sheet.pdf"},
		Position:          1,
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
		Title:          "Wall Faucet",
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
