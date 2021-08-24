package model

import (
	"context"

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

func RemovePushTokens(ctx context.Context, db *sqlx.DB, userIDs []int) error {
	if len(userIDs) == 0 {
		return nil
	}

	query := `UPDATE users
SET expo_push_token = null
WHERE id in (?)
`

	query, args, err := sqlx.In(query, userIDs)
	if err != nil {
		return err
	}

	_, err = db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	return nil
}
