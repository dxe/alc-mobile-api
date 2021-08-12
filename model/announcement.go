package model

import (
	"errors"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type Announcement struct {
	ID           int    `db:"id"`
	ConferenceID int    `db:"conference_id"`
	Title        string `db:"title"`
	Message      string `db:"message"`
	Icon         string `db:"icon"`
	CreatedBy    string `db:"created_by"`
	SendTime     string `db:"send_time"`
	Sent         bool   `db:"sent"`
}

type AnnouncementOptions struct {
	IncludeScheduled       bool
	ConvertTimeToUSPacific bool
}

func ListAnnouncements(db *sqlx.DB, options AnnouncementOptions) ([]Announcement, error) {
	timeQuery := "send_time"
	if options.ConvertTimeToUSPacific {
		timeQuery = `DATE_FORMAT(CONVERT_TZ(send_time, 'UTC','US/Pacific'), "%a, %b %e, %Y at %l:%i %p") as send_time`
	}

	query := `
SELECT id, conference_id, title, message, icon, created_by,
       ` + timeQuery + `,
       sent
FROM announcements
ORDER BY announcements.send_time asc
`
	if !options.IncludeScheduled {
		query += " AND sent = 1"
	}
	var announcements []Announcement
	if err := db.Select(&announcements, query); err != nil {
		return announcements, fmt.Errorf("failed to list announcements: %w", err)
	}
	if announcements == nil {
		announcements = make([]Announcement, 0)
	}
	return announcements, nil
}

func GetAnnouncementByID(db *sqlx.DB, id string) (Announcement, error) {
	const query = `
SELECT id, conference_id, title, message, icon, created_by, send_time, sent
FROM announcements
WHERE id = ?
`
	var announcements []Announcement
	if err := db.Select(&announcements, query, id); err != nil {
		return Announcement{}, fmt.Errorf("failed to select announcement: %w", err)
	}
	if len(announcements) == 0 {
		return Announcement{}, errors.New("found no announcements with given id")
	}
	return announcements[0], nil
}

func SaveAnnouncement(db *sqlx.DB, announcement Announcement) error {
	if announcement.ID == 0 {
		return insertAnnouncement(db, announcement)
	}
	return updateAnnouncement(db, announcement)
}

func insertAnnouncement(db *sqlx.DB, announcement Announcement) error {
	log.Println("inserting!")
	query := `
INSERT INTO announcements (conference_id, title, message, icon, created_by, send_time)
VALUES (:conference_id, TRIM(:title), TRIM(:message), :icon, :created_by, :send_time)
`
	if _, err := db.NamedExec(query, announcement); err != nil {
		return fmt.Errorf("failed to insert announcement: %w", err)
	}
	return nil
}

func updateAnnouncement(db *sqlx.DB, announcement Announcement) error {
	query := `
UPDATE announcements
SET conference_id = :conference_id, title = TRIM(:title), message = TRIM(:message), icon = :icon, created_by = :created_by, send_time = :send_time
WHERE id = :id
`
	if _, err := db.NamedExec(query, announcement); err != nil {
		return fmt.Errorf("failed to update announcement: %w", err)
	}
	return nil
}

func DeleteAnnouncement(db *sqlx.DB, id string) error {
	if id == "" {
		return errors.New("announcement id must be provided")
	}
	const query = "DELETE FROM announcements WHERE id = ?"
	res, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete announcement: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("failed to delete announcement: no rows affected")
	}
	return nil
}
