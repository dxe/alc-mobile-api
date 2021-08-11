package model

import (
	"log"

	"github.com/jmoiron/sqlx"
)

type Notification struct {
	UserID         int    `db:"user_id"`
	AnnouncementID int    `db:"announcement_id"`
	Status         string `db:"status"`
	Receipt        string `db:"receipt"`
	ReceiptStatus  string `db:"receipt_status"`
	Timestamp      string `db:"timestamp"`
}

func EnqueueAnnouncementNotifications(db *sqlx.DB) {
	log.Println("Starting to enqueue notifications for announcements.")

	tx := db.MustBegin()

	insertQuery := `
INSERT IGNORE into notifications (user_id, announcement_id, status)

SELECT users.id as user_id, announcements.id as announcement_id, "queued" as status
FROM announcements
JOIN users ON users.conference_id = announcements.conference_id
WHERE sent = 0 AND send_time >= NOW() AND expo_push_token is not null
ORDER BY send_time asc
`
	results := tx.MustExec(insertQuery)
	notificationRows, err := results.RowsAffected()
	if err != nil {
		tx.Rollback()
		panic(err)
	}

	updateQuery := `
UPDATE announcements
SET sent = 1
WHERE id in (SELECT DISTINCT announcement_id FROM notifications) AND sent = 0
`
	results = tx.MustExec(updateQuery)
	announcementRows, err := results.RowsAffected()
	if err != nil {
		tx.Rollback()
		panic(err)
	}

	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		panic(err)
	}

	log.Printf("Enqueued %d notifications for %d announcements.\n", notificationRows, announcementRows)
}
