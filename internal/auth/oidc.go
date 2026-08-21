package auth

import (
	"cmp"
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	sessionTTL = 24 * time.Hour
	loginTTL   = 10 * time.Minute
)

// Options configures an Authenticator.
type Options struct {
	// Issuer is the Authentik OIDC issuer URL.
	Issuer string
	// ClientID and ClientSecret identify this environment's Authentik
	// application. The flow is confidential-client plus PKCE.
	ClientID     string
	ClientSecret string
	// BaseURL is the site origin; the redirect URL is BaseURL/auth/callback.
	BaseURL string
	// Group is the Authentik group required to author.
	Group string
	// SessionKey seals the cookies.
	SessionKey []byte
}

// Authenticator runs the OIDC login flow and owns the session cookies.
// Provider discovery is lazy so the server boots (and public pages serve)
// while the identity provider is down or not yet configured.
type Authenticator struct {
	codec        *Codec
	issuer       string
	clientID     string
	clientSecret string
	redirectURL  string
	group        string
	secure       bool

	mu       sync.Mutex
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// New builds an Authenticator. It does not contact the issuer.
func New(opts Options) (*Authenticator, error) {
	codec, err := NewCodec(opts.SessionKey)
	if err != nil {
		return nil, err
	}
	return &Authenticator{
		codec:        codec,
		issuer:       opts.Issuer,
		clientID:     opts.ClientID,
		clientSecret: opts.ClientSecret,
		redirectURL:  strings.TrimRight(opts.BaseURL, "/") + "/auth/callback",
		group:        opts.Group,
		secure:       strings.HasPrefix(opts.BaseURL, "https://"),
	}, nil
}

// Group returns the required author group.
func (a *Authenticator) Group() string { return a.group }

// setup discovers the provider on first use, caching only success so a down
// issuer is retried on the next login attempt.
func (a *Authenticator) setup(ctx context.Context) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.oauth != nil {
		return a.oauth, a.verifier, nil
	}
	provider, err := oidc.NewProvider(ctx, a.issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc discovery: %w", err)
	}
	a.oauth = &oauth2.Config{
		ClientID:     a.clientID,
		ClientSecret: a.clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  a.redirectURL,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}
	a.verifier = provider.Verifier(&oidc.Config{ClientID: a.clientID})
	return a.oauth, a.verifier, nil
}

// Begin starts the login flow: it seals state and the PKCE verifier into the
// login cookie and redirects to Authentik. returnTo is where the callback
// lands afterwards; it must be a site-local path.
func (a *Authenticator) Begin(w http.ResponseWriter, r *http.Request, returnTo string) error {
	conf, _, err := a.setup(r.Context())
	if err != nil {
		return err
	}

	stateBytes := make([]byte, 16)
	if err := fillRandom(stateBytes); err != nil {
		return err
	}
	ls := loginState{
		State:    hex.EncodeToString(stateBytes),
		Verifier: oauth2.GenerateVerifier(),
		Return:   sanitizeReturn(returnTo),
		Expires:  time.Now().Add(loginTTL),
	}
	token, err := a.codec.Seal(ls)
	if err != nil {
		return err
	}
	a.setCookie(w, loginCookie, token, loginTTL)

	url := conf.AuthCodeURL(ls.State, oauth2.S256ChallengeOption(ls.Verifier))
	http.Redirect(w, r, url, http.StatusFound)
	return nil
}

// Complete finishes the callback: state check, code exchange with the PKCE
// verifier, ID token verification, and the session cookie write. It returns
// the site-local path the login started from.
func (a *Authenticator) Complete(w http.ResponseWriter, r *http.Request) (string, error) {
	conf, verifier, err := a.setup(r.Context())
	if err != nil {
		return "", err
	}

	cookie, err := r.Cookie(loginCookie)
	if err != nil {
		return "", fmt.Errorf("no login in progress")
	}
	var ls loginState
	if err := a.codec.Open(cookie.Value, &ls); err != nil {
		return "", fmt.Errorf("login cookie: %w", err)
	}
	a.clearCookie(w, loginCookie)
	if time.Now().After(ls.Expires) {
		return "", fmt.Errorf("login expired")
	}
	if r.URL.Query().Get("state") != ls.State {
		return "", fmt.Errorf("state mismatch")
	}

	token, err := conf.Exchange(r.Context(), r.URL.Query().Get("code"), oauth2.VerifierOption(ls.Verifier))
	if err != nil {
		return "", fmt.Errorf("code exchange: %w", err)
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		return "", fmt.Errorf("no id_token in token response")
	}
	idToken, err := verifier.Verify(r.Context(), rawID)
	if err != nil {
		return "", fmt.Errorf("verify id token: %w", err)
	}

	var claims struct {
		Name              string   `json:"name"`
		PreferredUsername string   `json:"preferred_username"`
		Email             string   `json:"email"`
		Groups            []string `json:"groups"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", fmt.Errorf("id token claims: %w", err)
	}

	session := Session{
		Subject: idToken.Subject,
		Name:    cmp.Or(claims.PreferredUsername, claims.Name, claims.Email),
		Groups:  claims.Groups,
		Expires: time.Now().Add(sessionTTL),
	}
	if err := a.WriteSession(w, session); err != nil {
		return "", err
	}
	return ls.Return, nil
}

// sanitizeReturn keeps the post-login redirect on this site: a path starting
// with a single slash. Anything else lands on /admin.
func sanitizeReturn(p string) string {
	if strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "//") {
		return p
	}
	return "/admin"
}
