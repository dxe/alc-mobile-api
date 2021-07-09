package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Announcement struct {
	ID        int       `db:"id" json:"id"`
	Title     string    `db:"title" json:"title"`
	Message   string    `db:"message" json:"message"`
	Icon      string    `db:"icon" json:"icon"`
	CreatedBy string    `db:"created_by" json:"created_by"`
	SendTime  time.Time `db:"send_time" json:"send_time"`
	Sent      bool      `db:"sent" json:"sent"`
}

type AnnouncementOptions struct {
	IncludeScheduled bool
	ConferenceID     int
}

func ListAnnouncements(db *sqlx.DB, options AnnouncementOptions) ([]Announcement, error) {
	if options.ConferenceID == 0 {
		return nil, errors.New("must provide conference id")
	}

	query := "SELECT id, title, message, icon, created_by, send_time, sent FROM announcements WHERE conference_id = ?"
	if !options.IncludeScheduled {
		query += " AND sent = 1"
	}
	var announcements []Announcement
	if err := db.Select(&announcements, query, options.ConferenceID); err != nil {
		return announcements, fmt.Errorf("failed to list announcements: %w", err)
	}
	if announcements == nil {
		return nil, errors.New("no announcements found")
	}
	return announcements, nil
}
