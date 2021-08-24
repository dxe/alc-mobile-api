package main

import (
	"encoding/json"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	expo "github.com/jakehobbs/exponent-server-sdk-golang/sdk"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/coreos/go-oidc"
	"github.com/dxe/alc-mobile-api/model"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
)

var (
	flagProd = flag.Bool("prod", false, "whether to run in production mode")
)

func config(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing configuration for %v", key)
	}
	return v
}

func configInt(key string) int {
	intVal, err := strconv.Atoi(config(key))
	if err != nil {
		log.Fatalf("failed to parse configuration for %v as int", key)
	}
	return intVal
}

const isoTimeLayout = "2006-01-02T15:04:05.000Z"
const dbTimeLayout = "2006-01-02 15:04:05"

// getDSN returns the DSN string for the backing MySQL database.
func getDSN() string {
	cfg, err := mysql.ParseDSN(config("DB_DSN"))
	if err != nil {
		log.Fatalf("failed to parse MySQL DSN: %v", err)
	}

	cfg.ParseTime = true
	cfg.Params = map[string]string{
		// TODO(mdempsky): Is this still necessary/appropriate? The MySQL
		// driver now recommends using the "collation" parameter instead,
		// which defaults to "utf8mb4_general_ci".
		//
		// See https://github.com/go-sql-driver/mysql#unicode-support.
		"charset": "utf8mb4",
	}
	return cfg.FormatDSN()
}

func main() {
	flag.Parse()

	// TODO(mdempsky): Generalize.
	mux := http.NewServeMux()

	db := model.NewDB(getDSN())

	// TODO: Consider not doing this each time the application loads.
	// It may be better to do it via a script instead.
	if !*flagProd {
		//model.WipeDatabase(db, *flagProd)
		//model.InitDatabase(db)
		//model.InsertMockData(db, *flagProd)
	}

	clientID := config("OAUTH_CLIENT_ID")
	clientSecret := config("OAUTH_CLIENT_SECRET")

	conf, verifier, err := newGoogleVerifier(clientID, clientSecret)
	if err != nil {
		log.Fatalf("failed to create Google OIDC verifier: %v", err)
	}

	awsRegion := config("S3_REGION")
	awsAuthID := config("S3_AUTH_ID")
	awsSecret := config("S3_SECRET")

	awsSession, err := NewAWSSession(awsRegion, awsAuthID, awsSecret)
	if err != nil {
		log.Fatalf("failed to create AWS session: %v", err)
	}

	expoPushClient := expo.NewPushClient(&expo.ClientConfig{AccessToken: os.Getenv("EXPO_PUSH_ACCESS_TOKEN")})

	newServer := func(w http.ResponseWriter, r *http.Request) *server {
		return &server{
			conf:           conf,
			verifier:       verifier,
			awsSession:     awsSession,
			expoPushClient: expoPushClient,

			db: db,
			w:  w,
			r:  r,
		}
	}

	handle := func(path string, method func(*server)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			method(newServer(w, r))
			log.Printf("Handled request %v in %v.", path, time.Since(start))
		})
	}

	// handleAuth is like handle, but it requires the user to be logged
	// in with OAuth2 credentials first. Currently, this means with an
	// @directactioneverywhere.com account, because our OAuth2 settings
	// are configured to "Internal".
	handleAuth := func(path string, method func(*server)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			s := newServer(w, r)

			if r.URL.Path != path {
				http.NotFound(w, r)
				return
			}

			email, err := s.googleEmail()
			if err != nil {
				s.redirect(absURL("/login"))
				return
			}
			s.email = email

			method(s)
		})
	}

	// Index & auth pages
	handleAuth("/", (*server).index)
	handle("/login", (*server).login)
	handle("/logout", (*server).logout)
	handle("/auth", (*server).auth)
	handleAuth("/admin", (*server).admin)

	// Admin conference pages
	handleAuth("/admin/conferences", (*server).adminConferences)
	handleAuth("/admin/conference/details", (*server).adminConferenceDetails)
	handleAuth("/admin/conference/save", (*server).adminConferenceSave)
	handleAuth("/admin/conference/delete", (*server).adminConferenceDelete)

	// Admin location pages
	handleAuth("/admin/locations", (*server).adminLocations)
	handleAuth("/admin/location/details", (*server).adminLocationDetails)
	handleAuth("/admin/location/save", (*server).adminLocationSave)
	handleAuth("/admin/location/delete", (*server).adminLocationDelete)

	// Admin event pages
	handleAuth("/admin/events", (*server).adminEvents)
	handleAuth("/admin/event/details", (*server).adminEventDetails)
	handleAuth("/admin/event/save", (*server).adminEventSave)
	handleAuth("/admin/event/delete", (*server).adminEventDelete)

	// Admin info pages
	handleAuth("/admin/info", (*server).adminInfo)
	handleAuth("/admin/info/details", (*server).adminInfoDetails)
	handleAuth("/admin/info/save", (*server).adminInfoSave)
	handleAuth("/admin/info/delete", (*server).adminInfoDelete)

	// Admin announcement pages
	handleAuth("/admin/announcements", (*server).adminAnnouncements)
	handleAuth("/admin/announcement/details", (*server).adminAnnouncementDetails)
	handleAuth("/admin/announcement/save", (*server).adminAnnouncementSave)
	handleAuth("/admin/announcement/delete", (*server).adminAnnouncementDelete)

	// Healthcheck for load balancer
	handle("/healthcheck", (*server).health)

	// Public API
	handle("/api/announcement/list", apiAnnouncementList.serve)
	handle("/api/conference/list", apiConferenceList.serve)
	handle("/api/event/list", apiEventList.serve)
	handle("/api/event/rsvp", apiEventRSVP.serve)
	handle("/api/info/list", apiInfoList.serve)
	handle("/api/user/add", apiUserAdd.serve)
	handle("/api/user/register_push_notifications", apiUserRegisterPushNotifications.serve)

	// Static file server
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Start go routines for queueing and sending notifications.
	go EnqueueAnnouncementNotificationsWrapper(db)
	go SendNotificationsWrapper(db, expoPushClient)

	log.Println("Server started. Listening on port 8080.")
	server := &http.Server{Addr: ":8080", Handler: mux}
	log.Fatal(server.ListenAndServe())
}

type server struct {
	conf           *oauth2.Config
	verifier       *oidc.IDTokenVerifier
	awsSession     *session.Session
	expoPushClient *expo.PushClient

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
		UserEmail           string
		PageName            string
		PageData            interface{}
		Conferences         []model.Conference
		DefaultConferenceID int
	}

	// TODO: Consider only doing getting this data on pages you need it.
	// Alternatively, have a Conference selector on the nav bar that is reflected on all pages.
	conferences, err := model.ListConferences(s.db, model.ConferenceOptions{})
	if err != nil {
		log.Println(err)
		panic("failed to get conferences")
	}

	data := templateData{
		UserEmail:           s.email,
		PageName:            name,
		PageData:            pageData,
		Conferences:         conferences,
		DefaultConferenceID: configInt("DEFAULT_CONFERENCE_ID"),
	}

	tmpl, err := template.New("").Funcs(template.FuncMap{
		"emailToName": func(email string) string {
			components := strings.Split(email, "@")
			return strings.Title(components[0])
		},
	}).ParseGlob("templates/*.html")
	if err != nil {
		log.Println(err)
		panic("failed to parse template")
	}
	if err := tmpl.ExecuteTemplate(s.w, name+".html", data); err != nil {
		log.Println(err)
		panic("failed to execute template")
	}
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
