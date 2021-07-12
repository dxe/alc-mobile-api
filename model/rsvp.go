package model

type RSVP struct {
	EventID   int  `db:"event_id" json:"event_id"`
	UserID    int  `db:"user_id" json:"user_id"`
	Attending bool `db:"attending" json:"attending"`
}

// TODO
