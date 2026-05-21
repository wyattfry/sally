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

func TestNormalizeMatchKey(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"K-3589", "k3589"},
		{"k3589", "k3589"},
		{"  Kohler ", "kohler"},
		{"Example Co.", "exampleco"},
		{"", ""},
	}
	for _, c := range cases {
		if got := normalizeMatchKey(c.in); got != c.want {
			t.Errorf("normalizeMatchKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestMothershipAPIDuplicateItemBlockedWithoutConfirm(t *testing.T) {
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
	user, err := q.CreateUser(ctx, queries.CreateUserParams{Email: "dup-test@example.com", Name: "Dup Test"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(ctx, queries.CreateProjectParams{OwnerUserID: user.ID, Name: "Dup Project"})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	schedA, err := q.CreateSchedule(ctx, queries.CreateScheduleParams{ProjectID: project.ID, Name: "Plumbing", Kind: "items", Position: 1})
	if err != nil {
		t.Fatalf("create sched A: %v", err)
	}
	schedB, err := q.CreateSchedule(ctx, queries.CreateScheduleParams{ProjectID: project.ID, Name: "Lighting", Kind: "items", Position: 2})
	if err != nil {
		t.Fatalf("create sched B: %v", err)
	}

	router := NewRouterWithDeps(config.Config{}, provider.NewStubExtractor(), web.Deps{
		Queries: q, DevUserEmail: "dup-test@example.com", DevUserName: "Dup Test",
	})

	// First item lands in schedule A.
	first := bytes.NewBufferString(`{"data":{"title":"Faucet","manufacturer":"Kohler","model_number":"K-3589"}}`)
	r1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedA.ID+"/items", first)
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(r1, req1)
	if r1.Code != http.StatusCreated {
		t.Fatalf("first insert: expected 201, got %d body=%s", r1.Code, r1.Body.String())
	}

	// Second attempt — same model, different casing/punctuation, into a different schedule
	// in the same project — must be flagged as a duplicate.
	second := bytes.NewBufferString(`{"data":{"title":"Same Faucet","manufacturer":"kohler","model_number":"k3589"}}`)
	r2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedB.ID+"/items", second)
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(r2, req2)
	if r2.Code != http.StatusConflict {
		t.Fatalf("dup attempt: expected 409, got %d body=%s", r2.Code, r2.Body.String())
	}
	var dup duplicateItemResponse
	if err := json.Unmarshal(r2.Body.Bytes(), &dup); err != nil {
		t.Fatalf("decode conflict body: %v", err)
	}
	if !dup.Duplicate || dup.ScheduleID != schedA.ID || dup.ScheduleName != "Plumbing" {
		t.Fatalf("unexpected conflict body: %#v", dup)
	}

	// With confirmDuplicate=true the insert should go through.
	third := bytes.NewBufferString(`{"data":{"title":"Same Faucet","manufacturer":"kohler","model_number":"k3589"},"confirmDuplicate":true}`)
	r3 := httptest.NewRecorder()
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/schedules/"+schedB.ID+"/items", third)
	req3.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(r3, req3)
	if r3.Code != http.StatusCreated {
		t.Fatalf("confirmed dup: expected 201, got %d body=%s", r3.Code, r3.Body.String())
	}
}

func TestMothershipAPIRoomRoundTrips(t *testing.T) {
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
		Email: "api-room-test@example.com",
		Name:  "API Room Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Room API Project",
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
		DevUserEmail: "api-room-test@example.com",
		DevUserName:  "API Room Test",
	})

	body := bytes.NewBufferString(`{
		"data": {
			"title": "Range Hood",
			"manufacturer": "Example Co.",
			"model_number": "RH-100",
			"finish": "Stainless"
		},
		"room": "Kitchen",
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
		Room string `json:"room"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if created.Room != "Kitchen" {
		t.Fatalf("expected room %q, got %q", "Kitchen", created.Room)
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
