package main

import (
	"fmt"
	"log"

	"github.com/dxe/alc-mobile-api/expo"

	"github.com/dxe/alc-mobile-api/model"

	"github.com/jmoiron/sqlx"
)

func NotificationsWorker(db *sqlx.DB) {
	log.Println("Notifications Worker started.")

	tx, err := db.Beginx()
	if err != nil {
		log.Println("Failed to begin database transaction.")
		return
	}

	notifications, err := model.SelectNotificationsToSend(tx)
	if err != nil {
		log.Printf(err.Error())
		tx.Rollback()
		return
	}

	if len(notifications) == 0 {
		log.Println("Found no queued notifications to send.")
		tx.Rollback()
		return
	}

	expoMessages := make([]expo.PushMessage, len(notifications))
	for i, n := range notifications {
		expoMessages[i] = expo.PushMessage{
			To:    n.ExpoPushToken,
			Title: n.Title,
			Body:  n.Body,
		}
	}

	expoResponses, err := expo.PublishMessages(expoMessages)
	if err != nil {
		log.Println(err.Error())
		tx.Rollback()
		return
	}

	for i, r := range expoResponses {
		fmt.Printf("%+v\n", r)
		err := model.UpdateNotificationStatus(tx, model.Notification{
			UserID:         notifications[i].UserID,
			AnnouncementID: notifications[i].AnnouncementID,
			Status:         r.Status,
			Receipt:        r.ID,
		})
		if err != nil {
			// There is no point in rolling back at this point, because the push notifications
			// have already been sent to Expo's API. Just log the error & continue.
			log.Println(err.Error())
		}
		if r.Details["error"] == expo.ErrorDeviceNotRegistered {
			if err := model.RemovePushToken(tx, notifications[i].UserID); err != nil {
				// If removing the token fails, log the error & continue. Next time a push
				// notification is attempted to be sent, it should make another attempt
				// to remove the token if the same error is encountered.
				log.Println(err)
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		fmt.Println("Failed to commit transaction.")
		tx.Rollback()
		return
	}
	log.Println("Notifications Worker finished.")
}
