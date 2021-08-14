package model

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

type Notification struct {
	// From the notifications table.
	ID              string `db:"id"`
	UserID          int    `db:"user_id"`
	AnnouncementID  int    `db:"announcement_id"`
	Status          string `db:"status"`
	LeaseExpiration int64  `db:"expiration_time"`
	Receipt         string `db:"receipt"`
	// From joined tables.
	ExpoPushToken string `db:"expo_push_token"`
	Title         string `db:"title"`
	Body          string `db:"body"`
}

func EnqueueAnnouncementNotifications(db *sqlx.DB) {
	log.Println("Starting to enqueue announcement notifications.")

	err := transact(db, func(tx *sqlx.Tx) error {
		insertQuery := `
INSERT IGNORE into notifications (user_id, announcement_id, status)
	SELECT users.id as user_id, announcements.id as announcement_id, "Queued" as status
	FROM announcements
	JOIN users ON users.conference_id = announcements.conference_id
	WHERE sent = 0 AND send_time <= NOW() AND expo_push_token like "ExponentPushToken[%]"
	ORDER BY send_time asc
`
		results := tx.MustExec(insertQuery)
		notificationRows, err := results.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to insert into notifications table: %w", err)
		}

		updateQuery := `
UPDATE announcements
SET sent = 1
WHERE id in (SELECT DISTINCT announcement_id FROM notifications) AND sent = 0
`
		results = tx.MustExec(updateQuery)
		announcementRows, err := results.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to update announcements table: %w", err)
		}

		log.Printf("Enqueued %d notifications for %d announcements.\n", notificationRows, announcementRows)

		return nil
	})

	if err != nil {
		log.Println("Failed to enqueue announcement notifications.")
		return
	}

}

func SelectNotificationsToSend(db *sqlx.DB) ([]Notification, error) {
	var notifications []Notification

	currentTime := time.Now().Unix()
	fiveMinFromNow := time.Now().Add(5 * time.Minute).Unix()

	err := transact(db, func(tx *sqlx.Tx) error {
		selectQuery := `
			SELECT
				CONCAT(user_id,"-",announcement_id) as id,
				notifications.user_id,
				notifications.announcement_id,
				expo_push_token,
				announcements.title,
				announcements.message as body
			FROM notifications
			JOIN users ON users.id = notifications.user_id
			JOIN announcements ON announcements.id = notifications.announcement_id
			WHERE
				notifications.status in ("Queued", "Leased")
				AND expo_push_token like "ExponentPushToken[%]"
				AND lease_expiration <= ?
			LIMIT 100
			FOR UPDATE SKIP LOCKED
		`

		if err := tx.Select(&notifications, selectQuery, currentTime); err != nil {
			return fmt.Errorf("select query failed: %w", err)
		}

		if len(notifications) == 0 {
			return nil
		}

		idsToUpdate := make([]string, len(notifications))
		for i, n := range notifications {
			idsToUpdate[i] = n.ID
		}

		updateQuery := `
			UPDATE notifications
				SET
					status = "Leased",
    				lease_expiration = ` + strconv.FormatInt(fiveMinFromNow, 10) + `
			WHERE CONCAT(user_id,"-",announcement_id) IN (?)
		`

		query, args, err := sqlx.In(updateQuery, idsToUpdate)
		if err != nil {
			return fmt.Errorf("failed to prepare query using IN clause: %w", err)
		}

		if _, err := tx.Query(query, args...); err != nil {
			return fmt.Errorf("update query failed: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to select notifications to send: %w", err)
	}

	return notifications, nil
}

func UpdateNotificationStatus(db *sqlx.DB, notifications []Notification) error {
	// It seems that sqlx doesn't let you use NamedExec with a slice of structs
	// when doing an UPDATE, so we will do an INSERT ... ON DUPLICATE KEY UPDATE instead.
	query := `
INSERT INTO notifications (user_id, announcement_id, status)
VALUES (:user_id, :announcement_id, :status)
ON DUPLICATE KEY UPDATE status=VALUES(status)
`
	_, err := db.NamedExec(query, notifications)
	if err != nil {
		return err
	}
	return nil
}
