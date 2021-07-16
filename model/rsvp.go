package model

type RSVP struct {
	EventID   int    `db:"event_id"`
	UserID    int    `db:"user_id"`
	Attending bool   `db:"attending"`
	Timestamp string `db:"timestamp"`
}

// TODO
