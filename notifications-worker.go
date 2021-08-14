package main

import (
	"fmt"
	"log"
	"time"

	"github.com/dxe/alc-mobile-api/model"
	"github.com/jmoiron/sqlx"
	expo "github.com/oliveroneill/exponent-server-sdk-golang/sdk"
)

func filterValidTokens(notifications []model.Notification) ([]model.Notification, []expo.PushMessage) {
	var validNotifications []model.Notification
	var messages []expo.PushMessage
	for _, n := range notifications {
		pushToken, err := expo.NewExponentPushToken(n.ExpoPushToken)
		if err != nil {
			continue
		}
		validNotifications = append(validNotifications, n)
		messages = append(messages, expo.PushMessage{
			To:    []expo.ExponentPushToken{pushToken},
			Title: n.Title,
			Body:  n.Body,
		})
	}
	return validNotifications, messages
}

const (
	StatusSent                = "Sent"
	StatusDeviceNotRegistered = "DeviceNotRegistered"
	StatusUnknownError        = "UnknownError"
)

func notificationsWorker(db *sqlx.DB, client *expo.PushClient) (err error) {
	notifications, err := model.SelectNotificationsToSend(db)
	if err != nil {
		return err
	}

	if len(notifications) == 0 {
		return nil
	}

	validNotifications, validExpoMessages := filterValidTokens(notifications)

	// TODO: use context to cancel if this takes too long?
	expoResponses, err := client.PublishMultiple(validExpoMessages)
	if err != nil {
		return fmt.Errorf("failed to publish messages via expo api: %w", err)
	}

	// Update each notification with the status.
	for i, r := range expoResponses {
		if r.Status == expo.SuccessStatus {
			validNotifications[i].Status = StatusSent
			continue
		}
		if r.Details["error"] == expo.ErrorDeviceNotRegistered {
			validNotifications[i].Status = StatusDeviceNotRegistered
			model.RemovePushToken(db, validNotifications[i].UserID)
			continue
		}
		validNotifications[i].Status = StatusUnknownError
	}

	// Write the new status to the database.
	err = model.UpdateNotificationStatus(db, validNotifications)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	return nil

}

func NotificationsWorkerWrapper(db *sqlx.DB, client *expo.PushClient) {
	for {
		log.Println("Notifications worker started.")
		if err := notificationsWorker(db, client); err != nil {
			log.Printf("Notifications worker failed: %v\n", err.Error())
		} else {
			log.Println("Notifications worker finished.")
		}
		time.Sleep(15 * time.Second)
	}
}
