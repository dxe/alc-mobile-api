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

func SendNotifications(db *sqlx.DB, client *expo.PushClient) (err error) {
	currentTime := time.Now()
	fiveMinFromNow := time.Now().Add(5 * time.Minute)

	notifications, err := model.SelectNotificationsToSend(db, currentTime, fiveMinFromNow)
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

	// Create a slice to store IDs of unregistered users to use to update
	// the database without having to iterate through all of the notifications
	// an extra time.
	var unregisteredUsers []int

	// Update each notification with the status.
	for i, r := range expoResponses {
		switch {
		case r.Status == expo.SuccessStatus:
			validNotifications[i].Status = StatusSent
		case r.Details["error"] == expo.ErrorDeviceNotRegistered:
			validNotifications[i].Status = StatusDeviceNotRegistered
			unregisteredUsers = append(unregisteredUsers, validNotifications[i].UserID)
		default:
			validNotifications[i].Status = StatusUnknownError
		}
	}

	// Write the new status to the database.
	err = model.UpdateNotificationStatus(db, validNotifications)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}

	// Remove tokens from users table for unregistered users.
	err = model.RemovePushTokens(db, unregisteredUsers)
	if err != nil {
		return fmt.Errorf("failed to update remove unregistered push tokens from users: %w", err)
	}

	return nil

}

func SendNotificationsWrapper(db *sqlx.DB, client *expo.PushClient) {
	for {
		log.Println("Notifications worker started.")
		if err := SendNotifications(db, client); err != nil {
			log.Printf("Notifications worker failed: %v\n", err.Error())
		} else {
			log.Println("Notifications worker finished.")
		}
		time.Sleep(15 * time.Second)
	}
}

func EnqueueAnnouncementNotificationsWrapper(db *sqlx.DB) {
	for {
		log.Println("Starting to enqueue announcement notifications.")
		if err := model.EnqueueAnnouncementNotifications(db); err != nil {
			log.Printf("Failed to enqueue announcementn notifications: %v\n", err.Error())
		} else {
			log.Println("Finished enqueuing announcement notifications.")
		}
		time.Sleep(60 * time.Second)
	}
}
