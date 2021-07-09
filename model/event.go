package model

import (
	"errors"
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
	ImageID     int       `db:"image_id" json:"image_id"`       // TODO(jhobbs): Handle this the same way?
}

// TODO(jhobbs): Be sure to use conference id.

func getAllEvents(db *sqlx.DB) ([]Event, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
