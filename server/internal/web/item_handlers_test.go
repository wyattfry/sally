package web

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	appdb "sally/server/internal/db"
	queries "sally/server/internal/db/generated"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestItemPagesCreateAndListItem(t *testing.T) {
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
		Email: "item-pages-test@example.com",
		Name:  "Item Pages Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Item Test Project " + time.Now().Format("150405.000000"),
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

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "item-pages-test@example.com",
		DevUserName:  "Item Pages Test",
	})

	form := url.Values{}
	form.Set("code", "B-01")
	form.Set("title", "Wall Faucet")
	form.Set("description", "Wall-mounted faucet with rough valve.")
	form.Set("manufacturer", "Example Co.")
	form.Set("model_number", "WF-200")
	form.Set("finish", "Polished Chrome")
	form.Set("notes", "Verify rough-in.")
	form.Set("source_url", "https://example.com/products/wf-200")

	path := "/projects/" + project.ID + "/schedules/" + schedule.ID + "/items"
	createReq := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createResp := httptest.NewRecorder()

	router.ServeHTTP(createResp, createReq)

	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected create to redirect with 303, got %d", createResp.Code)
	}
	location := createResp.Header().Get("Location")
	expectedLocation := "/projects/" + project.ID + "#schedule-" + schedule.ID
	if location != expectedLocation {
		t.Fatalf("expected redirect to %q, got %q", expectedLocation, location)
	}

	// Fragment is stripped by the router; GET the project page directly.
	showReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	showResp := httptest.NewRecorder()

	router.ServeHTTP(showResp, showReq)

	if showResp.Code != http.StatusOK {
		t.Fatalf("expected show status 200, got %d", showResp.Code)
	}
	body := showResp.Body.String()
	for _, expected := range []string{"B-01", "Wall Faucet", "Example Co.", "Polished Chrome", "Verify rough-in."} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected project page to include %q, got %s", expected, body)
		}
	}
}

func TestEditItemPageShowsFullBreadcrumb(t *testing.T) {
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
		Email: "item-breadcrumb-test@example.com",
		Name:  "Item Breadcrumb Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Item Breadcrumb Project " + time.Now().Format("150405.000000"),
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
		Description:       "",
		Manufacturer:      "Example Co.",
		ModelNumber:       "WF-200",
		Finish:            "Chrome",
		FinishModelNumber: "",
		Notes:             "",
		SourceUrl:         "",
		SourceTitle:       "",
		SourceImageUrl:    "",
		SourcePdfLinks:    []string{},
		Position:          1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "item-breadcrumb-test@example.com",
		DevUserName:  "Item Breadcrumb Test",
	})

	editPath := "/projects/" + project.ID + "/schedules/" + schedule.ID + "/items/" + item.ID + "/edit"
	req := httptest.NewRequest(http.MethodGet, editPath, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, project.Name) {
		t.Fatalf("expected edit-item breadcrumb to include project name %q, got:\n%s", project.Name, body)
	}
	if !strings.Contains(body, "Bath") {
		t.Fatalf("expected edit-item breadcrumb to include schedule name, got:\n%s", body)
	}
	if !strings.Contains(body, `/projects"`) {
		t.Fatalf("expected edit-item breadcrumb to include /projects link, got:\n%s", body)
	}
}

func TestItemPagesUpdateAndDeleteItem(t *testing.T) {
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
		Email: "item-update-test@example.com",
		Name:  "Item Update Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Item Update Project " + time.Now().Format("150405.000000"),
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
		Description:       "Wall-mounted faucet.",
		Manufacturer:      "Example Co.",
		ModelNumber:       "WF-200",
		Finish:            "Chrome",
		FinishModelNumber: "",
		Notes:             "",
		SourceUrl:         "",
		SourceTitle:       "",
		SourceImageUrl:    "",
		SourcePdfLinks:    []string{},
		Position:          1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "item-update-test@example.com",
		DevUserName:  "Item Update Test",
	})

	form := url.Values{}
	form.Set("code", "B-02")
	form.Set("title", "Updated Faucet")
	form.Set("description", "Updated description.")
	form.Set("manufacturer", "Updated Co.")
	form.Set("model_number", "WF-201")
	form.Set("finish", "Brushed Nickel")
	form.Set("notes", "Updated notes.")
	form.Set("position", "2")

	editPath := "/projects/" + project.ID + "/schedules/" + schedule.ID + "/items/" + item.ID + "/edit"
	updateReq := httptest.NewRequest(http.MethodPost, editPath, strings.NewReader(form.Encode()))
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateResp := httptest.NewRecorder()
	router.ServeHTTP(updateResp, updateReq)

	if updateResp.Code != http.StatusSeeOther {
		t.Fatalf("expected update to redirect with 303, got %d", updateResp.Code)
	}

	projectReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	projectResp := httptest.NewRecorder()
	router.ServeHTTP(projectResp, projectReq)
	for _, expected := range []string{"B-02", "Updated Faucet", "Updated Co.", "Brushed Nickel", "Updated notes."} {
		if !strings.Contains(projectResp.Body.String(), expected) {
			t.Fatalf("expected project page to include %q after update, got %s", expected, projectResp.Body.String())
		}
	}

	deleteReq := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/schedules/"+schedule.ID+"/items/"+item.ID+"/delete", nil)
	deleteResp := httptest.NewRecorder()
	router.ServeHTTP(deleteResp, deleteReq)
	if deleteResp.Code != http.StatusSeeOther {
		t.Fatalf("expected delete to redirect with 303, got %d", deleteResp.Code)
	}

	deletedResp := httptest.NewRecorder()
	router.ServeHTTP(deletedResp, projectReq)
	if strings.Contains(deletedResp.Body.String(), "Updated Faucet") {
		t.Fatalf("expected deleted item to be absent from project page, got %s", deletedResp.Body.String())
	}
}

func TestItemZoneAppearsOnProjectPage(t *testing.T) {
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
		Email: "item-zone-test@example.com",
		Name:  "Item Zone Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Zone Test Project " + time.Now().Format("150405.000000"),
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

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "item-zone-test@example.com",
		DevUserName:  "Item Zone Test",
	})

	// Create an item via the web form with a zone.
	form := url.Values{}
	form.Set("title", "Range Hood")
	form.Set("zone", "Kitchen")
	form.Set("manufacturer", "Example Co.")

	path := "/projects/" + project.ID + "/schedules/" + schedule.ID + "/items"
	createReq := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createResp := httptest.NewRecorder()
	router.ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", createResp.Code)
	}

	// Verify the zone header appears on the project page.
	showReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	showResp := httptest.NewRecorder()
	router.ServeHTTP(showResp, showReq)
	if showResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", showResp.Code)
	}
	body := showResp.Body.String()
	if !strings.Contains(body, "Range Hood") {
		t.Fatalf("expected item title in project page, got:\n%s", body)
	}
	if !strings.Contains(body, "Kitchen") {
		t.Fatalf("expected zone header on project page, got:\n%s", body)
	}
	if !strings.Contains(body, "zone-row") {
		t.Fatalf("expected zone-row CSS class on project page, got:\n%s", body)
	}
}

func TestEditItemPreservesSourceImageUrl(t *testing.T) {
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
		Email: "item-preserve-test@example.com",
		Name:  "Item Preserve Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: user.ID,
		Name:        "Preserve Test Project " + time.Now().Format("150405.000000"),
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
		ScheduleID:     schedule.ID,
		Title:          "Wall Faucet",
		SourceImageUrl: "https://example.com/faucet.jpg",
		SourceTitle:    "Wall Faucet Product Page",
		SourcePdfLinks: []string{"https://example.com/spec.pdf"},
		Position:       1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: "item-preserve-test@example.com",
		DevUserName:  "Item Preserve Test",
	})

	// Edit the item to add a zone, sending only the fields the edit form exposes.
	editPath := "/projects/" + project.ID + "/schedules/" + schedule.ID + "/items/" + item.ID + "/edit"

	// Simulate what the edit form sends: hidden fields for source_image_url etc.
	form := url.Values{}
	form.Set("title", "Wall Faucet")
	form.Set("zone", "Primary Bath")
	form.Set("source_image_url", item.SourceImageUrl)
	form.Set("source_title", item.SourceTitle)
	form.Set("source_pdf_links", strings.Join(item.SourcePdfLinks, "\n"))
	form.Set("position", "1")

	req := httptest.NewRequest(http.MethodPost, editPath, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d", resp.Code)
	}

	// The project page must still show the thumbnail after the zone edit.
	showReq := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	showResp := httptest.NewRecorder()
	router.ServeHTTP(showResp, showReq)
	body := showResp.Body.String()
	if !strings.Contains(body, "https://example.com/faucet.jpg") {
		t.Fatalf("expected source_image_url preserved after zone edit, got:\n%s", body)
	}
	if !strings.Contains(body, "Primary Bath") {
		t.Fatalf("expected zone header after edit, got:\n%s", body)
	}
}
