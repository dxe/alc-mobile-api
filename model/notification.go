package model

import (
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

type Notification struct {
	// From the notifications table.
	UserID         int    `db:"user_id"`
	AnnouncementID int    `db:"announcement_id"`
	Status         string `db:"status"`
	Receipt        string `db:"receipt"`
	// From joined tables.
	ExpoPushToken string `db:"expo_push_token"`
	Title         string `db:"title"`
	Body          string `db:"body"`
}

func EnqueueAnnouncementNotifications(db *sqlx.DB) {
	log.Println("Starting to enqueue announcement notifications.")

	tx := db.MustBegin()

	insertQuery := `
INSERT IGNORE into notifications (user_id, announcement_id, status)
	SELECT users.id as user_id, announcements.id as announcement_id, "queued" as status
	FROM announcements
	JOIN users ON users.conference_id = announcements.conference_id
	WHERE sent = 0 AND send_time <= NOW() AND expo_push_token is not null
	ORDER BY send_time asc
`
	results := tx.MustExec(insertQuery)
	notificationRows, err := results.RowsAffected()
	if err != nil {
		fmt.Println("failed to insert announcement notifications")
		tx.Rollback()
		return
	}

	updateQuery := `
UPDATE announcements
SET sent = 1
WHERE id in (SELECT DISTINCT announcement_id FROM notifications) AND sent = 0
`
	results = tx.MustExec(updateQuery)
	announcementRows, err := results.RowsAffected()
	if err != nil {
		fmt.Println("failed to update announcements table")
		tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println("failed to commit announcements to notifications table")
		tx.Rollback()
		return
	}

	log.Printf("Enqueued %d notifications for %d announcements.\n", notificationRows, announcementRows)
}

func SelectNotificationsToSend(tx *sqlx.Tx) ([]Notification, error) {
	query := `
SELECT notifications.user_id, notifications.announcement_id, expo_push_token, announcements.title, announcements.message as body
FROM notifications
JOIN users ON users.id = notifications.user_id
JOIN announcements on announcements.id = notifications.announcement_id
WHERE notifications.status = "queued" AND expo_push_token is not null
LIMIT 100
FOR UPDATE SKIP LOCKED
`

	var notifications []Notification
	if err := tx.Select(&notifications, query); err != nil {
		return nil, fmt.Errorf("failed to select notifications to send: %w", err)
	}

	return notifications, nil
}

func UpdateNotificationStatus(tx *sqlx.Tx, notification Notification) error {
	query := `UPDATE notifications
	   SET status = :status
	   WHERE user_id = :user_id and announcement_id = :announcement_id`
	_, err := tx.NamedExec(query, notification)
	if err != nil {
		return err
	}
	return nil
}
