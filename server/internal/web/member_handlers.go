package web

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	queries "sally/server/internal/db/generated"
)

func (a app) addProjectMember(w http.ResponseWriter, r *http.Request) {
	user, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.Form.Get("email")))
	if email == "" {
		http.Error(w, "email is required", http.StatusBadRequest)
		return
	}

	if email == strings.ToLower(user.Email) {
		http.Redirect(w, r, "/projects/"+project.ID+"?member_error=own", http.StatusSeeOther)
		return
	}

	invitee, err := a.queries.GetUserByEmail(r.Context(), email)
	if errors.Is(err, sql.ErrNoRows) {
		http.Redirect(w, r, "/projects/"+project.ID+"?member_error=notfound", http.StatusSeeOther)
		return
	}
	if err != nil {
		http.Error(w, "could not look up user", http.StatusInternalServerError)
		return
	}

	if err := a.queries.AddProjectMember(r.Context(), queries.AddProjectMemberParams{
		ProjectID:       project.ID,
		UserID:          invitee.ID,
		InvitedByUserID: user.ID,
	}); err != nil {
		http.Error(w, "could not add member", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID, http.StatusSeeOther)
}

// leaveProject lets the currently signed-in user remove themselves as a
// collaborator. Uses loadUserProject (not AsOwner) so members can call it.
// Owners cannot leave their own project — they must delete it instead.
func (a app) leaveProject(w http.ResponseWriter, r *http.Request) {
	user, project, ok := a.loadUserProject(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}
	if project.OwnerUserID == user.ID {
		http.Error(w, "project owners cannot leave their own project", http.StatusBadRequest)
		return
	}
	if err := a.queries.RemoveProjectMember(r.Context(), queries.RemoveProjectMemberParams{
		ProjectID: project.ID,
		UserID:    user.ID,
	}); err != nil {
		http.Error(w, "could not leave project", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func (a app) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	_, project, ok := a.loadUserProjectAsOwner(w, r, r.PathValue("projectID"))
	if !ok {
		return
	}

	memberUserID := r.PathValue("memberUserID")
	if err := a.queries.RemoveProjectMember(r.Context(), queries.RemoveProjectMemberParams{
		ProjectID: project.ID,
		UserID:    memberUserID,
	}); err != nil {
		http.Error(w, "could not remove member", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/projects/"+project.ID, http.StatusSeeOther)
}
