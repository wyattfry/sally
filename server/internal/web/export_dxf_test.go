package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	appdb "sally/server/internal/db"
	queries "sally/server/internal/db/generated"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestExportProjectDXF(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()
	if err := appdb.RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	q := queries.New(conn)
	user, _ := q.CreateUser(context.Background(), queries.CreateUserParams{Email: "dxf-test@example.com", Name: "DXF Test"})
	project, _ := q.CreateProject(context.Background(), queries.CreateProjectParams{OwnerUserID: user.ID, Name: "DXF Project"})
	schedule, _ := q.CreateSchedule(context.Background(), queries.CreateScheduleParams{ProjectID: project.ID, Name: "Toilet Schedule", Kind: "items", Position: 1})
	if err := seedColumns(context.Background(), q, schedule.ID, "general"); err != nil {
		t.Fatalf("seed columns: %v", err)
	}
	data, _ := json.Marshal(map[string]string{
		"code": "T-1", "manufacturer": "Kohler", "model_number": "K-3589",
		"finish": "Polished Chrome", "notes": "ADA comfort height.\nVerify rough-in.",
	})
	_, err = q.CreateScheduleItem(context.Background(), queries.CreateScheduleItemParams{
		ScheduleID: schedule.ID, Data: data, Room: "Master Bath",
		SourcePdfLinks: []string{}, SourceImageUrls: []string{}, Position: 1,
	})
	if err != nil {
		t.Fatalf("create item: %v", err)
	}

	router := http.NewServeMux()
	RegisterRoutes(router, Deps{Queries: q, DevUserEmail: "dxf-test@example.com", DevUserName: "DXF Test"})

	path := "/projects/" + project.ID + "/export.dxf"
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest(http.MethodGet, path, nil))

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, want := range []string{"AC1009", "LTYPE", "CONTINUOUS", "BLOCKS", "$MODEL_SPACE", "ENTITIES", "Toilet Schedule", "Kohler", "K-3589", "Master Bath", "ADA comfort height"} {
		if !strings.Contains(body, want) {
			t.Errorf("DXF output missing %q", want)
		}
	}
	if !strings.HasSuffix(strings.TrimSpace(body), "EOF") {
		t.Errorf("DXF output should end with EOF, got tail: %q", body[max(0, len(body)-40):])
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
