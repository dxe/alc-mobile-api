package model

type User struct {
	ID             int    `db:"id" json:"id"`
	ConferenceID   int    `db:"conference_id" json:"conference_id"`
	Name           string `db:"name" json:"name"`
	Email          string `db:"email" json:"email"`
	DeviceID       string `db:"device_id" json:"device_id"`
	DeviceName     string `db:"device_name" json:"device_name"`
	DevicePlatform string `db:"device_platform" json:"device_platform"`
}

type UserOptions struct {
	ConferenceID int
}

// TODO
