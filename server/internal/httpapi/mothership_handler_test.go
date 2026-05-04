package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"sally/server/internal/config"
	appdb "sally/server/internal/db"
	queries "sally/server/internal/db/generated"
	"sally/server/internal/provider"
	"sally/server/internal/web"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestMothershipAPISavesScheduleItem(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := appdb.RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	q := queries.New(conn)
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "api-test@example.com",
		Name:  "API Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "API Project",
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

	router := NewRouterWithDeps(config.Config{}, provider.NewStubExtractor(), web.Deps{
		Queries:      q,
		DevUserEmail: "api-test@example.com",
		DevUserName:  "API Test",
	})

	projectsResp := httptest.NewRecorder()
	router.ServeHTTP(projectsResp, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if projectsResp.Code != http.StatusOK {
		t.Fatalf("expected projects status 200, got %d", projectsResp.Code)
	}
	if !strings.Contains(projectsResp.Body.String(), "API Project") {
		t.Fatalf("expected projects response to include project, got %s", projectsResp.Body.String())
	}

	schedulesResp := httptest.NewRecorder()
	router.ServeHTTP(schedulesResp, httptest.NewRequest(http.MethodGet, "/api/v1/projects/"+project.ID+"/schedules", nil))
	if schedulesResp.Code != http.StatusOK {
		t.Fatalf("expected schedules status 200, got %d", schedulesResp.Code)
	}
	if !strings.Contains(schedulesResp.Body.String(), "Bath") {
		t.Fatalf("expected schedules response to include schedule, got %s", schedulesResp.Body.String())
	}

	body := bytes.NewBufferString(`{
		"code":"B-01",
		"title":"Wall Faucet",
		"description":"Wall-mounted faucet.",
		"manufacturer":"Example Co.",
		"modelNumber":"WF-200",
		"finish":"Polished Chrome",
		"finishModelNumber":"WF-200-PC",
		"notes":"Verify rough-in.",
		"sourceUrl":"https://example.com/products/wf-200",
		"sourceTitle":"Example Co. WF-200 Wall Faucet",
		"sourceImageUrl":"https://example.com/faucet.jpg",
		"sourcePdfLinks":["https://example.com/spec-sheet.pdf"]
	}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/items", body)
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		Title        string   `json:"title"`
		Manufacturer string   `json:"manufacturer"`
		SourcePDFLinks []string `json:"sourcePdfLinks"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created item: %v", err)
	}
	if created.Title != "Wall Faucet" || created.Manufacturer != "Example Co." || len(created.SourcePDFLinks) != 1 {
		t.Fatalf("unexpected created item: %#v", created)
	}
}

func TestMothershipAPIZoneRoundTrips(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer conn.Close()

	if err := appdb.RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	q := queries.New(conn)
	user, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "api-zone-test@example.com",
		Name:  "API Zone Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Zone API Project",
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	schedule, err := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{
		ProjectID: project.ID,
		Name:      "Appliance Schedule",
		Position:  1,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	router := NewRouterWithDeps(config.Config{}, provider.NewStubExtractor(), web.Deps{
		Queries:      q,
		DevUserEmail: "api-zone-test@example.com",
		DevUserName:  "API Zone Test",
	})

	body := bytes.NewBufferString(`{
		"title":"Range Hood",
		"zone":"Kitchen",
		"manufacturer":"Example Co.",
		"modelNumber":"RH-100",
		"finish":"Stainless",
		"sourcePdfLinks":[]
	}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/items", body)
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		Zone string `json:"zone"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Zone != "Kitchen" {
		t.Fatalf("expected zone %q, got %q", "Kitchen", created.Zone)
	}
}
