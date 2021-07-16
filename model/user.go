package model

type User struct {
	ID            int    `db:"id"`
	ConferenceID  int    `db:"conference_id"`
	Name          string `db:"name"`
	Email         string `db:"email"`
	DeviceID      string `db:"device_id"`
	DeviceName    string `db:"device_name"`
	Platform      string `db:"platform"`
	Timestamp     string `db:"timestamp"`
	ExpoPushToken string `db:"expo_push_token"`
}

type UserOptions struct {
	ConferenceID int
}

// TODO
