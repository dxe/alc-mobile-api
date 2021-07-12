package model

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Event struct {
	ID            int           `db:"id" json:"id"`
	ConferenceID  int           `db:"conference_id" json:"conference_id"`
	Name          string        `db:"name" json:"name"`
	Description   string        `db:"description" json:"description"`
	StartTime     string        `db:"start_time" json:"start_time"`
	Length        int           `db:"length" json:"length"`
	KeyEvent      bool          `db:"key_event" json:"key_event"`
	AttendeeCount int           `db:"attendee_count" json:"attendee_count"` // TODO(jhobbs): Get this data from database.
	LocationID    int           `db:"location_id" json:"location_id"`       // TODO(jhobbs): Maybe we should just stick the Location here?
	ImageID       sql.NullInt64 `db:"image_id" json:"image_id"`             // TODO(jhobbs): Maybe just use the Image here?
}

type EventOptions struct {
	ConferenceID           int
	ConvertTimeToUSPacific bool
}

func ListEvents(db *sqlx.DB, options EventOptions) ([]Event, error) {
	if options.ConferenceID == 0 {
		return nil, errors.New("must provide conference id")
	}

	timeQuery := "start_time"
	if options.ConvertTimeToUSPacific {
		timeQuery = `DATE_FORMAT(CONVERT_TZ(start_time, 'UTC','US/Pacific'), "%a, %b %e, %Y at %l:%i %p") as start_time`
	}

	query := `SELECT id, conference_id, name, description, ` + timeQuery + `, length, key_event, location_id, IFNULL(image_id, 0) as image_id
FROM events WHERE conference_id = ?
ORDER BY events.start_time asc
`

	var events []Event
	if err := db.Select(&events, query, options.ConferenceID); err != nil {
		return events, fmt.Errorf("failed to list events: %w", err)
	}
	if events == nil {
		events = make([]Event, 0)
	}
	return events, nil
}

func GetEventByID(db *sqlx.DB, id string) (Event, error) {
	const query = `
SELECT id, conference_id, name, description, start_time, length, key_event, location_id, image_id
FROM events
WHERE id = ?
`
	var events []Event
	if err := db.Select(&events, query, id); err != nil {
		return Event{}, fmt.Errorf("failed to select event: %w", err)
	}
	if len(events) == 0 {
		return Event{}, errors.New("found no conference with given id")
	}
	return events[0], nil
}

func SaveEvent(db *sqlx.DB, event Event) error {
	if event.ID == 0 {
		return insertEvent(db, event)
	}
	return updateEvent(db, event)
}

func insertEvent(db *sqlx.DB, event Event) error {
	query := `
INSERT INTO events (conference_id, name, description, start_time, length, key_event, location_id, image_id)
VALUES (:conference_id, :name, :description, :start_time, :length, :key_event, :location_id, :image_id)
`
	if _, err := db.NamedExec(query, event); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
}

func updateEvent(db *sqlx.DB, event Event) error {
	query := `
UPDATE events
SET conference_id = :conference_id, name = :name, description = :description, start_time = :start_time, length = :length,
    key_event = :key_event, location_id = :location_id, image_id = :image_id
WHERE id = :id
`
	if _, err := db.NamedExec(query, event); err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}
	return nil
}

func DeleteEvent(db *sqlx.DB, id string) error {
	if id == "" {
		return errors.New("event id must be provided")
	}
	const query = "DELETE FROM events WHERE id = ?"
	res, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("failed to delete event: no rows affected")
	}
	return nil
}
