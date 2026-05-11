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
		Kind:      "items",
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
		"data": {
			"title": "Wall Faucet",
			"manufacturer": "Example Co.",
			"model_number": "WF-200",
			"finish": "Polished Chrome",
			"notes": "Verify rough-in."
		},
		"sourceUrl": "https://example.com/products/wf-200",
		"sourceTitle": "Example Co. WF-200 Wall Faucet",
		"sourceImageUrl": "https://example.com/faucet.jpg",
		"sourcePdfLinks": ["https://example.com/spec-sheet.pdf"]
	}`)
	createResp := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedule.ID+"/items", body)
	createReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d body=%s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		Data           map[string]string `json:"data"`
		SourcePDFLinks []string          `json:"sourcePdfLinks"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created item: %v", err)
	}
	if created.Data["title"] != "Wall Faucet" || created.Data["manufacturer"] != "Example Co." || len(created.SourcePDFLinks) != 1 {
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
		Kind:      "items",
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
		"data": {
			"title": "Range Hood",
			"manufacturer": "Example Co.",
			"model_number": "RH-100",
			"finish": "Stainless"
		},
		"zone": "Kitchen",
		"sourcePdfLinks": []
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

func TestListProjectsIncludesSharedProjects(t *testing.T) {
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
	ctx := context.Background()

	owner, err := q.CreateUser(ctx, queries.CreateUserParams{Email: "owner-shared-test@example.com", Name: "Owner"})
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	member, err := q.CreateUser(ctx, queries.CreateUserParams{Email: "member-shared-test@example.com", Name: "Member"})
	if err != nil {
		t.Fatalf("create member: %v", err)
	}

	ownedProject, err := q.CreateProject(ctx, queries.CreateProjectParams{OwnerUserID: member.ID, Name: "Member Owned Project"})
	if err != nil {
		t.Fatalf("create owned project: %v", err)
	}
	sharedProject, err := q.CreateProject(ctx, queries.CreateProjectParams{OwnerUserID: owner.ID, Name: "Shared With Member"})
	if err != nil {
		t.Fatalf("create shared project: %v", err)
	}
	if err := q.AddProjectMember(ctx, queries.AddProjectMemberParams{
		ProjectID:       sharedProject.ID,
		UserID:          member.ID,
		InvitedByUserID: owner.ID,
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	router := NewRouterWithDeps(config.Config{}, provider.NewStubExtractor(), web.Deps{
		Queries:      q,
		DevUserEmail: "member-shared-test@example.com",
		DevUserName:  "Member",
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &projects); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	names := make(map[string]bool, len(projects))
	for _, p := range projects {
		names[p.Name] = true
	}
	if !names[ownedProject.Name] {
		t.Errorf("owned project %q missing from response", ownedProject.Name)
	}
	if !names[sharedProject.Name] {
		t.Errorf("shared project %q missing from response", sharedProject.Name)
	}
}
