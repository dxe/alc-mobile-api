package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
		model.WipeDatabase(db, *flagProd)
		model.InitDatabase(db)
		model.InsertMockData(db, *flagProd)
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

	newServer := func(w http.ResponseWriter, r *http.Request) *server {
		return &server{
			conf:       conf,
			verifier:   verifier,
			awsSession: awsSession,

			db: db,
			w:  w,
			r:  r,
		}
	}

	handle := func(path string, method func(*server)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			method(newServer(w, r))
		})
	}

	// handleAuth is like handle, but it requires the user to be logged
	// in with OAuth2 credentials first. Currently, this means with an
	// @directactioneverywhere.com account, because our OAuth2 settings
	// are configured to "Internal".
	handleAuth := func(path string, method func(*server)) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
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
	handle("/api/info/list", apiInfoList.serve)

	// Static file server
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	log.Println("Server started. Listening on port 8080.")
	server := &http.Server{Addr: ":8080", Handler: mux}
	log.Fatal(server.ListenAndServe())
}

type server struct {
	conf       *oauth2.Config
	verifier   *oidc.IDTokenVerifier
	awsSession *session.Session

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

	conferences, err := model.ListConferences(s.db, model.ConferenceOptions{})

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

func (s *server) admin() {
	s.renderTemplate("index", nil)
}

func (s *server) adminConferences() {
	conferenceData, err := model.ListConferences(s.db, model.ConferenceOptions{ConvertTimeToUSPacific: true})
	if err != nil {
		panic(err)
	}
	s.renderTemplate("conferences", conferenceData)
}

func (s *server) adminConferenceDetails() {
	id := s.r.URL.Query().Get("id")
	if id == "" {
		// Form to create a new conference
		s.renderTemplate("conference_details", model.Conference{})
		return
	}
	// Form to update an existing conference
	conference, err := model.GetConferenceByID(s.db, id)
	if err != nil {
		s.adminError(err)
		return
	}
	s.renderTemplate("conference_details", conference)
}

func (s *server) adminConferenceSave() {
	if err := s.r.ParseForm(); err != nil {
		s.adminError(err)
		return
	}

	id, err := strconv.Atoi(s.r.Form.Get("ID"))
	if err != nil {
		s.adminError(err)
		return
	}

	startTime, err := time.Parse(isoTimeLayout, s.r.Form.Get("StartDate"))
	if err != nil {
		s.adminError(errors.New("start time is invalid"))
		return
	}

	endTime, err := time.Parse(isoTimeLayout, s.r.Form.Get("EndDate"))
	if err != nil {
		s.adminError(errors.New("end time is invalid"))
		return
	}

	conference := model.Conference{
		ID:        id,
		Name:      s.r.Form.Get("Name"),
		StartDate: startTime.Format(dbTimeLayout),
		EndDate:   endTime.Format(dbTimeLayout),
	}
	// update the database
	if err := model.SaveConference(s.db, conference); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/conferences")
}

func (s *server) adminConferenceDelete() {
	id := s.r.URL.Query().Get("id")
	if err := model.DeleteConference(s.db, id); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/conferences")
}

func (s *server) adminLocations() {
	locationData, err := model.ListLocations(s.db)
	if err != nil {
		panic(err)
	}
	s.renderTemplate("locations", locationData)
}

func (s *server) adminLocationDetails() {
	id := s.r.URL.Query().Get("id")
	if id == "" {
		// Form to create a new location
		s.renderTemplate("location_details", model.Location{})
		return
	}
	// Form to update an existing location
	location, err := model.GetLocationByID(s.db, id)
	if err != nil {
		s.adminError(err)
		return
	}
	s.renderTemplate("location_details", location)
}

func (s *server) adminLocationSave() {
	if err := s.r.ParseForm(); err != nil {
		s.adminError(err)
		return
	}

	id, err := strconv.Atoi(s.r.Form.Get("ID"))
	if err != nil {
		s.adminError(err)
		return
	}

	// parse floats
	lat, err := strconv.ParseFloat(s.r.Form.Get("Lat"), 64)
	if err != nil {
		s.adminError(err)
		return
	}
	lng, err := strconv.ParseFloat(s.r.Form.Get("Lng"), 64)
	if err != nil {
		s.adminError(err)
		return
	}

	location := model.Location{
		ID:      id,
		Name:    s.r.Form.Get("Name"),
		PlaceID: s.r.Form.Get("PlaceID"),
		Address: s.r.Form.Get("Address"),
		City:    s.r.Form.Get("City"),
		Lat:     lat,
		Lng:     lng,
	}
	// update the database
	if err := model.SaveLocation(s.db, location); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/locations")
}

func (s *server) adminLocationDelete() {
	id := s.r.URL.Query().Get("id")
	if err := model.DeleteLocation(s.db, id); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/locations")
}

func (s *server) adminEvents() {
	eventData, err := model.ListEvents(s.db, model.EventOptions{ConferenceID: 1, ConvertTimeToUSPacific: true})
	if err != nil {
		panic(err)
	}
	s.renderTemplate("events", eventData)
}

func (s *server) adminEventDetails() {
	locations, err := model.ListLocations(s.db)
	if err != nil {
		s.adminError(fmt.Errorf("failed to load locations: %w", err))
		return
	}

	id := s.r.URL.Query().Get("id")
	if id == "" {
		// Form to create a new event
		s.renderTemplate("event_details", map[string]interface{}{
			"Event":     model.Event{},
			"Locations": locations,
		})
		return
	}
	// Form to update an existing event
	event, err := model.GetEventByID(s.db, id)
	if err != nil {
		s.adminError(err)
		return
	}
	s.renderTemplate("event_details", map[string]interface{}{
		"Event":     event,
		"Locations": locations,
	})
}

func (s *server) adminEventSave() {
	maxImgSize := int64(1024 * 1000 * 5) // allow only 5MB of file size
	if err := s.r.ParseMultipartForm(maxImgSize); err != nil {
		s.adminError(fmt.Errorf("failed to parse form (image over 5MB?): %w", err))
		return
	}

	// TODO(jhobbs): make a CleanEventFormData function in the Event model to abstract some of this out of main.go?

	id, err := strconv.Atoi(s.r.Form.Get("ID"))
	if err != nil {
		s.adminError(err)
		return
	}

	conferenceID, err := strconv.Atoi(s.r.Form.Get("ConferenceID"))
	if err != nil {
		s.adminError(err)
		return
	}

	startTime, err := time.Parse(isoTimeLayout, s.r.Form.Get("StartTime"))
	if err != nil {
		s.adminError(errors.New("start time is invalid"))
		return
	}

	locationID, err := strconv.Atoi(s.r.Form.Get("LocationID"))
	if err != nil {
		s.adminError(err)
		return
	}

	var keyEvent bool
	if s.r.Form.Get("KeyEvent") == "on" {
		keyEvent = true
	}

	length, err := strconv.Atoi(s.r.Form.Get("Length"))
	if err != nil {
		s.adminError(err)
		return
	}

	var imageURL sql.NullString

	file, fileHeader, err := s.r.FormFile("Image")
	switch err {
	case nil:
		defer file.Close()
		// Handle the new file upload
		log.Println("Processing new file!")
		// TODO(jhobbs): get the full url to the file instead of just the name?
		imageURL.String, err = UploadFileToS3(s.awsSession, file, fileHeader)
		if err != nil {
			s.adminError(fmt.Errorf("failed to upload file: %w", err))
			return
		}
	case http.ErrMissingFile:
		// No file provided, so just use the existing URL
		log.Println("No file provided!")
		imageURL.String = s.r.Form.Get("ImageURL")
	default:
		// Unexpected error
		s.adminError(fmt.Errorf("failed to get uploaded file: %w", err))
		return
	}

	if imageURL.String != "" {
		imageURL.Valid = true
	}

	event := model.Event{
		ID:           id,
		ConferenceID: conferenceID,
		Name:         s.r.Form.Get("Name"),
		Description:  s.r.Form.Get("Description"),
		StartTime:    startTime.Format(dbTimeLayout),
		Length:       length,
		KeyEvent:     keyEvent,
		LocationID:   locationID,
		ImageURL:     imageURL,
	}

	// update the database
	if err := model.SaveEvent(s.db, event); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/events")
}

func (s *server) adminEventDelete() {
	id := s.r.URL.Query().Get("id")
	if err := model.DeleteEvent(s.db, id); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/events")
}

func (s *server) adminInfo() {
	infoData, err := model.ListInfo(s.db)
	if err != nil {
		panic(err)
	}
	s.renderTemplate("info", infoData)
}

func (s *server) adminInfoDetails() {
	id := s.r.URL.Query().Get("id")
	if id == "" {
		// Form to create a new info
		s.renderTemplate("info_details", model.Info{})
		return
	}
	// Form to update an existing event
	info, err := model.GetInfoByID(s.db, id)
	if err != nil {
		s.adminError(err)
		return
	}
	s.renderTemplate("info_details", info)
}

func (s *server) adminInfoSave() {
	if err := s.r.ParseForm(); err != nil {
		s.adminError(err)
		return
	}

	id, err := strconv.Atoi(s.r.Form.Get("ID"))
	if err != nil {
		s.adminError(err)
		return
	}

	displayOrder, err := strconv.Atoi(s.r.Form.Get("DisplayOrder"))
	if err != nil {
		s.adminError(err)
		return
	}

	info := model.Info{
		ID:           id,
		Title:        s.r.Form.Get("Title"),
		Subtitle:     s.r.Form.Get("Subtitle"),
		Content:      s.r.Form.Get("Content"),
		Icon:         s.r.Form.Get("Icon"),
		DisplayOrder: displayOrder,
	}

	// update the database
	if err := model.SaveInfo(s.db, info); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/info")
}

func (s *server) adminInfoDelete() {
	id := s.r.URL.Query().Get("id")
	if err := model.DeleteInfo(s.db, id); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/info")
}

func (s *server) adminAnnouncements() {
	announcementData, err := model.ListAnnouncements(s.db, model.AnnouncementOptions{
		ConferenceID:           1,
		IncludeScheduled:       true,
		ConvertTimeToUSPacific: true,
	})
	if err != nil {
		panic(err)
	}
	s.renderTemplate("announcements", announcementData)
}

func (s *server) adminAnnouncementDetails() {
	id := s.r.URL.Query().Get("id")
	if id == "" {
		// Form to create a new announcement
		s.renderTemplate("announcement_details", model.Announcement{})
		return
	}
	// Form to update an existing announcement
	announcement, err := model.GetAnnouncementByID(s.db, id)
	if err != nil {
		s.adminError(err)
		return
	}
	s.renderTemplate("announcement_details", announcement)
}

func (s *server) adminAnnouncementSave() {
	if err := s.r.ParseForm(); err != nil {
		s.adminError(err)
		return
	}

	id, err := strconv.Atoi(s.r.Form.Get("ID"))
	if err != nil {
		s.adminError(err)
		return
	}

	conferenceID, err := strconv.Atoi(s.r.Form.Get("ConferenceID"))
	if err != nil {
		s.adminError(err)
		return
	}

	sendTime, err := time.Parse(isoTimeLayout, s.r.Form.Get("SendTime"))
	if err != nil {
		s.adminError(errors.New("send time is invalid"))
		return
	}

	announcement := model.Announcement{
		ID:           id,
		ConferenceID: conferenceID,
		Title:        s.r.Form.Get("Title"),
		Message:      s.r.Form.Get("Message"),
		Icon:         s.r.Form.Get("Icon"),
		CreatedBy:    s.email,
		SendTime:     sendTime.Format(dbTimeLayout),
	}

	// update the database
	if err := model.SaveAnnouncement(s.db, announcement); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/announcements")
}

func (s *server) adminAnnouncementDelete() {
	id := s.r.URL.Query().Get("id")
	if err := model.DeleteAnnouncement(s.db, id); err != nil {
		s.adminError(err)
		return
	}
	s.redirect("/admin/announcements")
}

func (s *server) adminError(err error) {
	s.renderTemplate("error", err.Error())
}

func (s *server) listConferences() {
	s.serveJSON(model.ListConferences(s.db, model.ConferenceOptions{}))
}

func (s *server) listInfo() {
	s.serveJSON(model.ListInfo(s.db))
}

func (s *server) listAnnouncements() {
	conferenceID, err := strconv.Atoi(s.r.URL.Query().Get("conference_id"))
	if err != nil {
		s.serveJSON(nil, errors.New("conference_id missing or invalid"))
		return
	}
	s.serveJSON(model.ListAnnouncements(s.db, model.AnnouncementOptions{
		IncludeScheduled: false,
		ConferenceID:     conferenceID,
	}))
}

func (s *server) listEvents() {
	conferenceID, err := strconv.Atoi(s.r.URL.Query().Get("conference_id"))
	if err != nil {
		s.serveJSON(nil, errors.New("conference_id missing or invalid"))
		return
	}
	s.serveJSON(model.ListEvents(s.db, model.EventOptions{
		ConferenceID: conferenceID,
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
