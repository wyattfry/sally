package web

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	queries "sally/server/internal/db/generated"
)

// tryCurrentUser does a best-effort session → user lookup. Returns nil
// when oauth is disabled (dev mode), the session is absent, or any DB
// error occurs — callers should treat nil as "anonymous / not loaded".
func (a app) tryCurrentUser(r *http.Request) *queries.User {
	if a.queries == nil {
		return nil
	}
	if a.oauthConfig == nil {
		// Dev mode: synthesise a placeholder so the header still renders.
		u := &queries.User{Email: a.devUserEmail, Name: a.devUserName}
		return u
	}
	email, ok := getSessionEmail(r, a.sessionSecret)
	if !ok {
		return nil
	}
	user, err := a.queries.GetUserByEmail(r.Context(), email)
	if errors.Is(err, sql.ErrNoRows) || err != nil {
		return nil
	}
	return &user
}

// userInitials returns up to two characters suitable for an avatar badge.
// Prefers first letter of first + last name word; falls back to first two
// letters of email local-part.
func userInitials(u *queries.User) string {
	if u == nil {
		return "?"
	}
	name := strings.TrimSpace(u.Name)
	if name != "" {
		parts := strings.Fields(name)
		r0, _ := utf8.DecodeRuneInString(parts[0])
		if len(parts) == 1 {
			return strings.ToUpper(string(r0))
		}
		r1, _ := utf8.DecodeRuneInString(parts[len(parts)-1])
		return strings.ToUpper(string(r0) + string(r1))
	}
	// Fall back to email local-part.
	local := strings.SplitN(u.Email, "@", 2)[0]
	if len(local) >= 2 {
		r0, sz := utf8.DecodeRuneInString(local)
		r1, _ := utf8.DecodeRuneInString(local[sz:])
		return strings.ToUpper(string(r0) + string(r1))
	}
	if len(local) == 1 {
		return strings.ToUpper(local)
	}
	return "?"
}

// render buffers the template output and writes it only on success so
// template errors don't produce a half-written 200 with a 500 banner
// appended. A per-request template clone injects the current user so the
// header can render the avatar menu without struct changes on page data.
func (a app) render(w http.ResponseWriter, r *http.Request, data any) {
	t, err := pageTemplate.Clone()
	if err != nil {
		log.Printf("render: clone: %v", err)
		http.Error(w, "could not render page", http.StatusInternalServerError)
		return
	}
	user := a.tryCurrentUser(r)
	t.Funcs(template.FuncMap{
		"currentUser":     func() *queries.User { return user },
		"userInitials":    func() string { return userInitials(user) },
		"userDisplayName": func() string {
			if user == nil {
				return ""
			}
			if user.Name != "" {
				return user.Name
			}
			return user.Email
		},
	})
	// Re-parse after Funcs so the new functions are available in the templates.
	if _, err = t.ParseFS(templatesFS, "templates/page.html", "templates/partials/*.html", "templates/pages/*.html"); err != nil {
		log.Printf("render: reparse: %v", err)
		http.Error(w, "could not render page", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err = t.ExecuteTemplate(&buf, "page.html", data); err != nil {
		log.Printf("render: template execute: %v", err)
		http.Error(w, "could not render page", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

func (a app) renderNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	a.render(w, r, notFoundPage{Kind: "not-found", Title: "Page not found"})
}

func firstNonEmpty(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func firstPositive(value int, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func requestBaseURL(r *http.Request) string {
	scheme := "https"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if r.TLS == nil {
		scheme = "http"
	}
	return scheme + "://" + r.Host
}

func splitLines(value string) []string {
	lines := strings.Split(value, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseInt32(value string, fallback int32) int32 {
	var parsed int
	if _, err := fmt.Sscanf(strings.TrimSpace(value), "%d", &parsed); err != nil {
		return fallback
	}
	return int32(parsed)
}
