package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dxe/alc-mobile-api/model"
)

func (s *server) admin() {
	users, err := model.GetUserCount(s.db)
	if err != nil {
		s.adminError(err)
		return
	}
	s.renderTemplate("index", users)
}

func (s *server) adminConferences() {
	conferenceData, err := model.ListConferences(s.db, model.ConferenceOptions{ConvertTimeToUSPacific: true})
	if err != nil {
		s.adminError(err)
		return
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
	eventData, err := model.ListEvents(s.db, model.EventOptions{ConvertTimeToUSPacific: true})
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
		image, err := ResizeJPG(file, 1200)
		if err != nil {
			s.adminError(fmt.Errorf("failed to resize image: %w", err))
			return
		}
		imageURL.String, err = UploadFileToS3(s.awsSession, image, fileHeader.Filename)
		if err != nil {
			s.adminError(fmt.Errorf("failed to upload file: %w", err))
			return
		}
	case http.ErrMissingFile:
		// No file provided, so just use the existing URL
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
