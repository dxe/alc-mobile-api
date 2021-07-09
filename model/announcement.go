package model

import (
	"errors"
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

// TODO(jhobbs): Be sure to use conference id.

func getAllAnnouncements(db *sqlx.DB) ([]Announcement, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
