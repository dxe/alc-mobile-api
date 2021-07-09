package model

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type RSVP struct {
	EventID   int  `db:"event_id" json:"event_id"`
	UserID    int  `db:"user_id" json:"user_id"`
	Attending bool `db:"attending" json:"attending"`
}

func ListRSVPs(db *sqlx.DB) ([]RSVP, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
