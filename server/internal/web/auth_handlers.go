package web

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	queries "sally/server/internal/db/generated"
	"sally/server/internal/share"

	"golang.org/x/oauth2"
)

func (a app) loginPage(w http.ResponseWriter, r *http.Request) {
	if a.oauthConfig == nil {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}
	if _, ok := getSessionEmail(r, a.sessionSecret); ok {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}
	render(w, signInPage{Kind: "login", Title: "Sign in"})
}

func (a app) aboutPage(w http.ResponseWriter, r *http.Request) {
	render(w, aboutPage{Kind: "about", Title: "About Sally"})
}

func (a app) pressPage(w http.ResponseWriter, r *http.Request) {
	render(w, staticPage{Kind: "press", Title: "Press"})
}

func (a app) privacyPage(w http.ResponseWriter, r *http.Request) {
	render(w, staticPage{Kind: "privacy", Title: "Privacy Policy"})
}

func (a app) contactPage(w http.ResponseWriter, r *http.Request) {
	render(w, staticPage{Kind: "contact", Title: "Contact"})
}

func (a app) startGoogleOAuth(w http.ResponseWriter, r *http.Request) {
	if a.oauthConfig == nil {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}
	state, err := newOAuthState()
	if err != nil {
		http.Error(w, "could not generate state", http.StatusInternalServerError)
		return
	}
	setOAuthStateCookie(w, state)
	if r.URL.Query().Get("next") == "done" {
		setPostAuthCookie(w, "done")
	}
	http.Redirect(w, r, a.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOnline), http.StatusSeeOther)
}

func (a app) oauthCallback(w http.ResponseWriter, r *http.Request) {
	if a.oauthConfig == nil {
		http.Redirect(w, r, "/projects", http.StatusSeeOther)
		return
	}

	stateCookie, err := r.Cookie(oauthStateCookieName)
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		http.Error(w, "invalid oauth state", http.StatusBadRequest)
		return
	}
	clearOAuthStateCookie(w)

	token, err := a.oauthConfig.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		log.Printf("oauth exchange error: %v", err)
		http.Error(w, "oauth exchange failed", http.StatusBadGateway)
		return
	}

	email, name, err := googleUserInfo(r.Context(), a.oauthConfig, token)
	if err != nil {
		log.Printf("google userinfo error: %v", err)
		http.Error(w, "could not get user info", http.StatusBadGateway)
		return
	}

	if _, err := a.queries.CreateUser(context.Background(), queries.CreateUserParams{
		Email: email,
		Name:  name,
	}); err != nil {
		http.Error(w, "could not upsert user", http.StatusInternalServerError)
		return
	}

	setSessionCookie(w, a.sessionSecret, email)

	postAuth := "/projects"
	if c, err := r.Cookie(postAuthCookieName); err == nil && c.Value == "done" {
		postAuth = "/auth/done"
	}
	clearPostAuthCookie(w)
	http.Redirect(w, r, postAuth, http.StatusSeeOther)
}

func (a app) authDone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!doctype html>
<html><head><title>Signed in</title></head>
<body>
<p>Signed in! This window will close automatically.</p>
<script>window.close();</script>
</body></html>`))
}

func (a app) logout(w http.ResponseWriter, r *http.Request) {
	clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (a app) tokenLogin(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("t")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	lt, err := a.queries.GetLoginTokenByHash(r.Context(), share.HashToken(token))
	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "invalid or expired login link", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "could not validate token", http.StatusInternalServerError)
		return
	}

	user, err := a.queries.GetUser(r.Context(), lt.UserID)
	if err != nil {
		http.Error(w, "user not found", http.StatusInternalServerError)
		return
	}

	if err := a.queries.MarkLoginTokenUsed(r.Context(), lt.ID); err != nil {
		log.Printf("warning: could not mark login token used: %v", err)
	}

	setSessionCookie(w, a.sessionSecret, user.Email)
	http.Redirect(w, r, "/projects", http.StatusSeeOther)
}

func googleUserInfo(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (email, name string, err error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("userinfo status %d", resp.StatusCode)
	}
	var info struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", "", err
	}
	if info.Email == "" {
		return "", "", fmt.Errorf("empty email in userinfo response")
	}
	return info.Email, info.Name, nil
}
