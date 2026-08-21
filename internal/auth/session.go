package auth

import (
	"crypto/rand"
	"net/http"
	"slices"
	"time"
)

// Cookie names. The session cookie is the whole session; the login cookie
// exists only for the redirect round trip to Authentik.
const (
	sessionCookie = "codepuke_session"
	loginCookie   = "codepuke_login"
)

// Session is the sealed content of the session cookie.
type Session struct {
	Subject string    `json:"sub"`
	Name    string    `json:"name"`
	Groups  []string  `json:"groups"`
	Expires time.Time `json:"exp"`
}

// Valid reports whether the session is unexpired.
func (s Session) Valid() bool { return time.Now().Before(s.Expires) }

// InGroup reports whether the session carries the group claim.
func (s Session) InGroup(group string) bool { return slices.Contains(s.Groups, group) }

// loginState rides the login cookie across the redirect to Authentik: the
// CSRF state, the PKCE verifier, and where to land afterwards.
type loginState struct {
	State    string    `json:"state"`
	Verifier string    `json:"verifier"`
	Return   string    `json:"return"`
	Expires  time.Time `json:"exp"`
}

func fillRandom(b []byte) error {
	_, err := rand.Read(b)
	return err
}

func (a *Authenticator) setCookie(w http.ResponseWriter, name, value string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Authenticator) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ReadSession returns the request's valid session, if any.
func (a *Authenticator) ReadSession(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return Session{}, false
	}
	var s Session
	if err := a.codec.Open(cookie.Value, &s); err != nil || !s.Valid() {
		return Session{}, false
	}
	return s, true
}

// WriteSession seals s into the session cookie.
func (a *Authenticator) WriteSession(w http.ResponseWriter, s Session) error {
	token, err := a.codec.Seal(s)
	if err != nil {
		return err
	}
	a.setCookie(w, sessionCookie, token, time.Until(s.Expires))
	return nil
}

// Logout drops the session cookie.
func (a *Authenticator) Logout(w http.ResponseWriter) {
	a.clearCookie(w, sessionCookie)
}
