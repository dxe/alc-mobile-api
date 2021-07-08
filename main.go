package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

var (
	flagProd = flag.Bool("prod", false, "whether to run in production mode")
)

func config(key string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	log.Fatalf("missing configuration for %v", key)
	panic("unreachable")
}

func main() {
	// TODO(mdempsky): Generalize.
	r := http.DefaultServeMux

	// TODO(mdempsky): Initialize with a connection to a real DB.
	var db *sqlx.DB

	clientID := os.Getenv("OAUTH_CLIENT_ID")
	clientSecret := os.Getenv("OAUTH_CLIENT_SECRET")

	conf, verifier, err := newGoogleVerifier(clientID, clientSecret)
	if err != nil {
		log.Fatalf("failed to create Google OIDC verifier: %v", err)
	}

	handle := func(path string, method func(*server)) {
		r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			method(&server{conf, verifier, db, w, r})
		})
	}

	handle("/", (*server).index)
	handle("/login", (*server).login)
	handle("/auth", (*server).auth)
	handle("/healthcheck", (*server).health)

	log.Println("Server started. Listening on port 8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type server struct {
	conf     *oauth2.Config
	verifier *oidc.IDTokenVerifier

	db *sqlx.DB
	w  http.ResponseWriter
	r  *http.Request
}

func (s *server) index() {
	email, err := s.googleEmail()
	if err != nil {
		s.redirect(absURL("/login"))
		return
	}

	s.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(s.w, "Hello, %s\n", email)
	if isAdmin(email) {
		fmt.Fprintf(s.w, "(Psst, you're an admin.)\n")
	}
}

func (s *server) health() {
	s.w.WriteHeader(http.StatusOK)
	s.w.Write([]byte("OK"))
}

func (s *server) error(err error) {
	s.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(s.w, err)
}

func (s *server) redirect(dest string) {
	http.Redirect(s.w, s.r, dest, http.StatusFound)
}

func absURL(path string) string {
	// TODO(mdempsky): Use URL relative path resolution here? Or add a
	// flag to control the base URL?
	base := "http://localhost:8080"
	if *flagProd {
		base = "https://alc-mobile-api.dxe.io"
	}
	return base + path
}
