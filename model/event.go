package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Event struct {
	ID          int       `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	StartTime   time.Time `db:"start_time" json:"start_time"`
	Length      float32   `db:"length" json:"length"`
	KeyEvent    bool      `db:"key_event" json:"key_event"`
	LocationID  int       `db:"location_id" json:"location_id"` // TODO(jhobbs): Maybe we should just stick the Location here?
	ImageID     int       `db:"image_id" json:"image_id"`       // TODO(jhobbs): Handle this the same way? Handle nulls?
}

type EventOptions struct {
	ConferenceID int
}

func ListEvents(db *sqlx.DB, options EventOptions) ([]Event, error) {
	if options.ConferenceID == 0 {
		return nil, errors.New("must provide conference id")
	}

	query := `SELECT id, name, description, start_time, length, key_event, location_id, IFNULL(image_id, 0) as image_id
FROM events WHERE conference_id = ?`

	var events []Event
	if err := db.Select(&events, query, options.ConferenceID); err != nil {
		return events, fmt.Errorf("failed to list events: %w", err)
	}
	if events == nil {
		return nil, errors.New("no events found")
	}
	return events, nil
}
