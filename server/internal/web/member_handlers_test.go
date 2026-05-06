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

func newMemberTestRouter(t *testing.T, ownerEmail string) (*queries.Queries, http.Handler) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	if err := appdb.RunMigrations(context.Background(), conn, "../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	q := queries.New(conn)
	router := http.NewServeMux()
	RegisterRoutes(router, Deps{
		Queries:      q,
		DevUserEmail: ownerEmail,
		DevUserName:  "Member Test Owner",
	})
	return q, router
}

func createMemberTestProject(t *testing.T, q *queries.Queries, ownerEmail string) (queries.User, queries.Project) {
	t.Helper()
	owner, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: ownerEmail,
		Name:  "Member Test Owner",
	})
	if err != nil {
		t.Fatalf("create owner: %v", err)
	}
	project, err := q.CreateProject(context.Background(), queries.CreateProjectParams{
		OwnerUserID: owner.ID,
		Name:        "Member Test Project " + time.Now().Format("150405.000000"),
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	return owner, project
}

func TestAddProjectMember_Success(t *testing.T) {
	ownerEmail := "member-owner-add@example.com"
	q, router := newMemberTestRouter(t, ownerEmail)
	owner, project := createMemberTestProject(t, q, ownerEmail)

	invitee, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "member-invitee-add@example.com",
		Name:  "Invitee",
	})
	if err != nil {
		t.Fatalf("create invitee: %v", err)
	}

	form := url.Values{}
	form.Set("email", invitee.Email)
	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/members", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect, got %d: %s", resp.Code, resp.Body.String())
	}

	members, err := q.ListProjectMembersWithUser(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(members))
	}
	if members[0].UserID != invitee.ID {
		t.Errorf("expected member user ID %s, got %s", invitee.ID, members[0].UserID)
	}
	if members[0].InvitedByUserID != owner.ID {
		t.Errorf("expected invited_by %s, got %s", owner.ID, members[0].InvitedByUserID)
	}
}

func TestAddProjectMember_UnknownEmail(t *testing.T) {
	ownerEmail := "member-owner-unknown@example.com"
	q, router := newMemberTestRouter(t, ownerEmail)
	_, project := createMemberTestProject(t, q, ownerEmail)

	form := url.Values{}
	form.Set("email", "nobody@example.com")
	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/members", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", resp.Code)
	}
	if !strings.Contains(resp.Header().Get("Location"), "member_error=notfound") {
		t.Errorf("expected notfound error in redirect, got %s", resp.Header().Get("Location"))
	}
}

func TestAddProjectMember_OwnEmail(t *testing.T) {
	ownerEmail := "member-owner-self@example.com"
	q, router := newMemberTestRouter(t, ownerEmail)
	_, project := createMemberTestProject(t, q, ownerEmail)

	form := url.Values{}
	form.Set("email", ownerEmail)
	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/members", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d", resp.Code)
	}
	if !strings.Contains(resp.Header().Get("Location"), "member_error=own") {
		t.Errorf("expected own error in redirect, got %s", resp.Header().Get("Location"))
	}
}

func TestRemoveProjectMember(t *testing.T) {
	ownerEmail := "member-owner-remove@example.com"
	q, router := newMemberTestRouter(t, ownerEmail)
	owner, project := createMemberTestProject(t, q, ownerEmail)

	invitee, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "member-invitee-remove@example.com",
		Name:  "Invitee Remove",
	})
	if err != nil {
		t.Fatalf("create invitee: %v", err)
	}
	if err := q.AddProjectMember(context.Background(), queries.AddProjectMemberParams{
		ProjectID:       project.ID,
		UserID:          invitee.ID,
		InvitedByUserID: owner.ID,
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/members/"+invitee.ID+"/remove", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusSeeOther {
		t.Fatalf("expected 303, got %d: %s", resp.Code, resp.Body.String())
	}

	members, err := q.ListProjectMembersWithUser(context.Background(), project.ID)
	if err != nil {
		t.Fatalf("list members: %v", err)
	}
	if len(members) != 0 {
		t.Errorf("expected 0 members after removal, got %d", len(members))
	}
}

func TestMemberCanViewProject(t *testing.T) {
	ownerEmail := "member-owner-view@example.com"
	q, _ := newMemberTestRouter(t, ownerEmail)
	owner, project := createMemberTestProject(t, q, ownerEmail)

	invitee, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "member-invitee-view@example.com",
		Name:  "Invitee View",
	})
	if err != nil {
		t.Fatalf("create invitee: %v", err)
	}
	if err := q.AddProjectMember(context.Background(), queries.AddProjectMemberParams{
		ProjectID:       project.ID,
		UserID:          invitee.ID,
		InvitedByUserID: owner.ID,
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	// Build a router authenticated as the invitee (dev mode, no OAuth).
	databaseURL := os.Getenv("DATABASE_URL")
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	memberRouter := http.NewServeMux()
	RegisterRoutes(memberRouter, Deps{
		Queries:      q,
		DevUserEmail: invitee.Email,
		DevUserName:  invitee.Name,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.ID, nil)
	resp := httptest.NewRecorder()
	memberRouter.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("member expected 200 on GET project, got %d", resp.Code)
	}
	if !strings.Contains(resp.Body.String(), project.Name) {
		t.Errorf("expected project name in response, got: %s", resp.Body.String())
	}
}

func TestMemberCannotDeleteProject(t *testing.T) {
	ownerEmail := "member-owner-del@example.com"
	q, _ := newMemberTestRouter(t, ownerEmail)
	owner, project := createMemberTestProject(t, q, ownerEmail)

	invitee, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "member-invitee-del@example.com",
		Name:  "Invitee Del",
	})
	if err != nil {
		t.Fatalf("create invitee: %v", err)
	}
	if err := q.AddProjectMember(context.Background(), queries.AddProjectMemberParams{
		ProjectID:       project.ID,
		UserID:          invitee.ID,
		InvitedByUserID: owner.ID,
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	memberRouter := http.NewServeMux()
	RegisterRoutes(memberRouter, Deps{
		Queries:      q,
		DevUserEmail: invitee.Email,
		DevUserName:  invitee.Name,
	})

	req := httptest.NewRequest(http.MethodPost, "/projects/"+project.ID+"/delete", nil)
	resp := httptest.NewRecorder()
	memberRouter.ServeHTTP(resp, req)

	if resp.Code == http.StatusSeeOther {
		t.Fatal("member must not be able to delete the project (got redirect as if success)")
	}
	// Confirm project still exists.
	if _, err := q.GetProject(context.Background(), project.ID); err != nil {
		t.Errorf("project should still exist after member delete attempt: %v", err)
	}
}

func TestSharedProjectsAppearsOnProjectsList(t *testing.T) {
	ownerEmail := "member-owner-list@example.com"
	q, _ := newMemberTestRouter(t, ownerEmail)
	owner, project := createMemberTestProject(t, q, ownerEmail)

	invitee, err := q.CreateUser(context.Background(), queries.CreateUserParams{
		Email: "member-invitee-list@example.com",
		Name:  "Invitee List",
	})
	if err != nil {
		t.Fatalf("create invitee: %v", err)
	}
	if err := q.AddProjectMember(context.Background(), queries.AddProjectMemberParams{
		ProjectID:       project.ID,
		UserID:          invitee.ID,
		InvitedByUserID: owner.ID,
	}); err != nil {
		t.Fatalf("add member: %v", err)
	}

	databaseURL := os.Getenv("DATABASE_URL")
	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	inviteeRouter := http.NewServeMux()
	RegisterRoutes(inviteeRouter, Deps{
		Queries:      q,
		DevUserEmail: invitee.Email,
		DevUserName:  invitee.Name,
	})

	req := httptest.NewRequest(http.MethodGet, "/projects", nil)
	resp := httptest.NewRecorder()
	inviteeRouter.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	body := resp.Body.String()
	if !strings.Contains(body, project.Name) {
		t.Errorf("expected shared project %q to appear on invitee's /projects page", project.Name)
	}
	if !strings.Contains(body, "Shared with me") {
		t.Errorf("expected 'Shared with me' section heading in page body")
	}
}
