package model

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type User struct {
	ID             int    `db:"id" json:"id"`
	Name           string `db:"name" json:"name"`
	Email          string `db:"email" json:"email"`
	DeviceID       string `db:"device_id" json:"device_id"`
	DeviceName     string `db:"device_name" json:"device_name"`
	DevicePlatform string `db:"device_platform" json:"device_platform"`
}

// TODO(jhobbs): Be sure to use conference id.

func ListUsers(db *sqlx.DB) ([]User, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
