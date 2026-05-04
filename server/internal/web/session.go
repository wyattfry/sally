package web

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strings"
)

const sessionCookieName = "sally_session"
const oauthStateCookieName = "sally_oauth_state"

func signedCookieValue(secret []byte, value string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(value))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return value + "|" + sig
}

func verifySignedCookieValue(secret []byte, signed string) (string, bool) {
	idx := strings.LastIndex(signed, "|")
	if idx < 0 {
		return "", false
	}
	value := signed[:idx]
	expected := signedCookieValue(secret, value)
	if !hmac.Equal([]byte(signed), []byte(expected)) {
		return "", false
	}
	return value, true
}

func setSessionCookie(w http.ResponseWriter, secret []byte, email string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signedCookieValue(secret, email),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 60 * 60,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

func getSessionEmail(r *http.Request, secret []byte) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	return verifySignedCookieValue(secret, cookie.Value)
}

// GetSessionEmail is the exported form of getSessionEmail for use by other packages.
func GetSessionEmail(r *http.Request, secret []byte) (string, bool) {
	return getSessionEmail(r, secret)
}

// ValidateSessionToken validates a raw signed session value (e.g. from a header).
func ValidateSessionToken(secret []byte, token string) (string, bool) {
	return verifySignedCookieValue(secret, token)
}

func newOAuthState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func setOAuthStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
}

func clearOAuthStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     oauthStateCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

const postAuthCookieName = "sally_post_auth"

func setPostAuthCookie(w http.ResponseWriter, destination string) {
	http.SetCookie(w, &http.Cookie{
		Name:     postAuthCookieName,
		Value:    destination,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
}

func clearPostAuthCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     postAuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}
