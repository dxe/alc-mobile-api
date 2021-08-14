package model

import (
	"log"

	"github.com/jmoiron/sqlx"
)

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

func GetUserCount(db *sqlx.DB) (interface{}, error) {
	var results []struct {
		TotalUsers                   int `db:"total_users"`
		PushNotificationEnabledUsers int `db:"push_notification_enabled_users"`
	}
	if err := db.Select(&results, `
select
(select count(*) from users) as total_users,
(select count(*) from users where expo_push_token is not null) as push_notification_enabled_users
`); err != nil {
		return 0, err
	}
	return results[0], nil
}

func RemovePushToken(db *sqlx.DB, userID int) {
	query := `UPDATE users
SET expo_push_token = null
WHERE id = ?
`
	_, err := db.Exec(query, userID)
	if err != nil {
		log.Printf("failed to remove push token from user id %d: %v", userID, err.Error())
	}

	query = `UPDATE notifications
SET status = "DeviceNotRegistered"
WHERE user_id = ?
`
	_, err = db.Exec(query, userID)
	if err != nil {
		log.Printf("failed update notification status for user id %d: %v", userID, err.Error())
	}
}
