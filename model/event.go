package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/jmoiron/sqlx"
)

type Event struct {
	ID              int            `db:"id"`
	ConferenceID    int            `db:"conference_id"`
	Name            string         `db:"name"`
	Description     string         `db:"description"`
	StartTime       string         `db:"start_time"`
	Length          int            `db:"length"`
	KeyEvent        bool           `db:"key_event"`
	BreakoutSession bool           `db:"breakout_session"`
	LocationID      int            `db:"location_id"`
	ImageURL        sql.NullString `db:"image_url"`
}

type EventOptions struct {
	ConvertTimeToUSPacific bool
	ConferenceId           int
}

func ListEvents(db *sqlx.DB, options EventOptions) ([]Event, error) {
	timeQuery := "start_time"
	if options.ConvertTimeToUSPacific {
		timeQuery = `DATE_FORMAT(CONVERT_TZ(start_time, 'UTC','US/Pacific'), "%a, %b %e, %Y at %l:%i %p") as start_time`
	}
	whereClause := `WHERE conference_id = ` + strconv.Itoa(options.ConferenceId)

	// TODO(jhobbs): Join the Location table to provide full Location information.
	query := `SELECT id, conference_id, name, description, ` + timeQuery + `, length, key_event, breakout_session, location_id, image_url
FROM events ` + whereClause + `
ORDER BY events.start_time asc
`

	var events []Event
	if err := db.Select(&events, query); err != nil {
		return events, fmt.Errorf("failed to list events: %w", err)
	}
	if events == nil {
		events = make([]Event, 0)
	}
	return events, nil
}

func GetEventByID(db *sqlx.DB, id string) (Event, error) {
	const query = `
SELECT id, conference_id, name, description, start_time, length, key_event, breakout_session, location_id, image_url
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
INSERT INTO events (conference_id, name, description, start_time, length, key_event, breakout_session, location_id, image_url)
VALUES (:conference_id, TRIM(:name), TRIM(:description), :start_time, :length, :key_event, :breakout_session, :location_id, :image_url)
`
	if _, err := db.NamedExec(query, event); err != nil {
		return fmt.Errorf("failed to insert event: %w", err)
	}
	return nil
}

func updateEvent(db *sqlx.DB, event Event) error {
	query := `
UPDATE events
SET conference_id = :conference_id, name = TRIM(:name), description = TRIM(:description), start_time = :start_time, length = :length,
    key_event = :key_event, breakout_session = :breakout_session, location_id = :location_id, image_url = :image_url
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
