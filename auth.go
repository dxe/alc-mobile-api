package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc"
	"golang.org/x/oauth2"
)

// Cookie names.
const (
	cookieIDToken   = "api_id_token"
	cookieAuthState = "api_auth_state"
)

func newGoogleVerifier(clientID, clientSecret string) (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	provider, err := oidc.NewProvider(context.Background(), "https://accounts.google.com")
	if err != nil {
		return nil, nil, err
	}
	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     provider.Endpoint(),
		RedirectURL:  absURL("/auth"),
		Scopes:       []string{"email"},
	}
	verifier := provider.Verifier(&oidc.Config{
		ClientID: clientID,
	})
	return conf, verifier, nil
}

func isAdmin(email string) bool {
	// TODO(mdempsky): Use adb_users instead?
	return email == "matthew@dempsky.org" ||
		strings.HasSuffix(email, "@directactioneverywhere.com")
}

func (s *server) googleEmail() (string, error) {
	c, err := s.r.Cookie(cookieIDToken)
	if err != nil {
		return "", err
	}

	token, err := s.verifier.Verify(s.r.Context(), c.Value)
	if err != nil {
		return "", err
	}

	var claims struct {
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	err = token.Claims(&claims)
	if err != nil {
		return "", err
	}

	if !claims.EmailVerified {
		return "", errors.New("email not verified")
	}

	return claims.Email, nil
}

func (s *server) login() {
	state, err := nonce()
	if err != nil {
		s.error(err)
		return
	}
	http.SetCookie(s.w, &http.Cookie{
		Name:     cookieAuthState,
		Value:    state,
		MaxAge:   3600,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})

	var opts []oauth2.AuthCodeOption
	if s.r.URL.Query()["force"] != nil {
		// If the user is currently only signed into one
		// Google Account, we need to set
		// prompt=select_account to force the account chooser
		// dialog to appear. Otherwise, Google will just
		// redirect back to us again immediately.
		opts = append(opts, oauth2.SetAuthURLParam("prompt", "select_account"))
	}

	s.redirect(s.conf.AuthCodeURL(state, opts...))
}

func (s *server) auth() {
	c, err := s.r.Cookie(cookieAuthState)
	if err != nil {
		s.error(err)
		return
	}
	if c.Value != s.r.FormValue("state") {
		s.error(errors.New("state mismatch"))
		return
	}

	token, err := s.conf.Exchange(s.r.Context(), s.r.FormValue("code"))
	if err != nil {
		s.error(err)
		return
	}

	idToken := token.Extra("id_token").(string)
	http.SetCookie(s.w, &http.Cookie{
		Name:     cookieIDToken,
		Value:    idToken,
		MaxAge:   3600,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	s.redirect(absURL("/"))
}

// nonce returns a 256-bit random hex string.
func nonce() (string, error) {
	var buf [32]byte
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
