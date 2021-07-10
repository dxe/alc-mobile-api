package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/dxe/alc-mobile-api/model"

	"github.com/coreos/go-oidc"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"

	_ "github.com/go-sql-driver/mysql"
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
	flag.Parse()

	// TODO(mdempsky): Generalize.
	r := http.DefaultServeMux

	connectionString := config("DB_USER") + ":" + config("DB_PASSWORD") +
		"@" + config("DB_PROTOCOL") + "/" + config("DB_NAME") +
		"?parseTime=true&charset=utf8mb4"
	if *flagProd {
		connectionString += "&tls=true"
	}
	db := model.NewDB(connectionString)

	clientID := config("OAUTH_CLIENT_ID")
	clientSecret := config("OAUTH_CLIENT_SECRET")

	conf, verifier, err := newGoogleVerifier(clientID, clientSecret)
	if err != nil {
		log.Fatalf("failed to create Google OIDC verifier: %v", err)
	}

	newServer := func(w http.ResponseWriter, r *http.Request) *server {
		return &server{
			conf:     conf,
			verifier: verifier,

			db: db,
			w:  w,
			r:  r,
		}
	}

	handle := func(path string, method func(*server)) {
		r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			method(newServer(w, r))
		})
	}

	// handleAuth is like handle, but it requires the user to be logged
	// in with OAuth2 credentials first. Currently, this means with an
	// @directactioneverywhere.com account, because our OAuth2 settings
	// are configured to "Internal".
	handleAuth := func(path string, method func(*server)) {
		r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			s := newServer(w, r)

			email, err := s.googleEmail()
			if err != nil {
				s.redirect(absURL("/login"))
				return
			}
			s.email = email

			method(s)
		})
	}

	// Admin pages
	handleAuth("/", (*server).index)
	handle("/login", (*server).login)
	handle("/auth", (*server).auth)
	handleAuth("/admin", (*server).admin)
	handleAuth("/admin/conferences", (*server).adminConferences)

	// Healthcheck page for load balancer
	handle("/healthcheck", (*server).health)

	// Unauthed API
	handle("/conference/list", (*server).listConferences)
	handle("/info/list", (*server).listInfo)
	handle("/announcement/list", (*server).listAnnouncements)
	handle("/event/list", (*server).listEvents)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Println("Server started. Listening on port 8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

type server struct {
	conf     *oauth2.Config
	verifier *oidc.IDTokenVerifier

	email string

	db *sqlx.DB
	w  http.ResponseWriter
	r  *http.Request
}

func (s *server) index() {
	s.w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if s.r.URL.Path != "/" {
		http.NotFound(s.w, s.r)
		return
	}
	s.redirect(absURL("/admin"))
}

func (s *server) renderTemplate(name string, pageData interface{}) {
	type templateData struct {
		UserEmail string
		PageData  interface{}
	}
	data := templateData{
		UserEmail: s.email,
		PageData:  pageData,
	}

	tmpl, err := template.New("").ParseGlob("templates/*.html")
	if err != nil {
		panic("failed to parse template")
	}
	if err := tmpl.ExecuteTemplate(s.w, name, data); err != nil {
		panic("failed to execute template")
	}
}

func (s *server) admin() {
	s.renderTemplate("index.html", nil)
}

func (s *server) adminConferences() {
	conferenceData, err := model.ListConferences(s.db)
	// TODO(jhobbs): Consider not returning an error if no conferences are found to make this more simple.
	if err != nil && err.Error() != "no conferences found" {
		panic(err)
	}
	s.renderTemplate("conferences.html", conferenceData)
}

func (s *server) listConferences() {
	s.serveJSON(model.ListConferences(s.db))
}

func (s *server) listInfo() {
	s.serveJSON(model.ListInfo(s.db))
}

func (s *server) listAnnouncements() {
	// TODO: pass in the conference id as a parameter
	s.serveJSON(model.ListAnnouncements(s.db, model.AnnouncementOptions{
		IncludeScheduled: false,
		ConferenceID:     1,
	}))
}

func (s *server) listEvents() {
	// TODO: pass in the conference id as a parameter
	s.serveJSON(model.ListEvents(s.db, model.EventOptions{
		ConferenceID: 1,
	}))
}

func (s *server) health() {
	s.serveJSON("OK", nil)
}

func (s *server) serveJSON(data interface{}, err error) {
	if err != nil {
		s.writeJSON(map[string]string{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	s.w.WriteHeader(http.StatusOK)
	s.writeJSON(map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}

func (s *server) writeJSON(v interface{}) {
	s.w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(s.w)
	err := enc.Encode(v)
	if err != nil {
		log.Printf("Error writing JSON: %v", err.Error())
	}
}

func (s *server) redirect(dest string) {
	http.Redirect(s.w, s.r, dest, http.StatusFound)
}

func absURL(path string) string {
	// TODO(mdempsky): Use URL relative path resolution here?
	return config("BASE_URL") + path
}
