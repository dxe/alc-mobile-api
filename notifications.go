package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dxe/alc-mobile-api/model"

	"github.com/jmoiron/sqlx"
)

type ExpoNotification struct {
	UserID         int    `db:"user_id" json:"-"`
	AnnouncementID int    `db:"announcement_id" json:"-"`
	To             string `json:"to"`
	Title          string `json:"title"`
	Body           string `json:"body"`
}

type ExpoTickets struct {
	Data []struct {
		Status  string   // error or ok
		ID      string   // receipt id
		Message string   // error message
		Details struct { // error details
			Error string // DeviceNotRegistered
		}
	}
	Errors []struct {
		Code    string
		Message string
	}
}

func SendNotifications(notifications []ExpoNotification) ExpoTickets {
	path := "https://exp.host/--/api/v2/push/send"

	var reqBody bytes.Buffer
	err := json.NewEncoder(&reqBody).Encode(notifications)
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", path, &reqBody)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("EXPO_PUSH_ACCESS_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var respBody ExpoTickets
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(respBody.Errors[0].Message)
		panic(errors.New("Expo returned status code " + strconv.Itoa(resp.StatusCode)))
	}

	return respBody
}

type ExpoReceipts struct {
	Data map[string]struct {
		Status  string   // error or ok
		Message string   // error message
		Details struct { // error details
			Error string // DeviceNotRegistered, MessageTooBig, MessageRateExceeded
		}
	}
	Errors []struct {
		Code    string
		Message string // PUSH_TOO_MANY_EXPERIENCE_IDS, PUSH_TOO_MANY_NOTIFICATIONS, PUSH_TOO_MANY_RECEIPTS
	}
}

func GetNotificationReceiptStatus(IDs []string) ExpoReceipts {
	path := "https://exp.host/--/api/v2/push/getReceipts"

	var reqBody bytes.Buffer
	err := json.NewEncoder(&reqBody).Encode(struct{ ids []string }{ids: IDs})
	if err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", path, &reqBody)
	if err != nil {
		panic(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("EXPO_PUSH_ACCESS_TOKEN"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	var respBody ExpoReceipts
	err = json.NewDecoder(resp.Body).Decode(&respBody)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Println(respBody.Errors[0].Message)
		panic(errors.New("Expo returned status code " + strconv.Itoa(resp.StatusCode)))
	}

	return respBody
}

func SendNotificationsWorker(db *sqlx.DB) {
	log.Println("Started SendNotifications worker.")
	// select 100 for update
	tx := db.MustBegin()

	query := `
SELECT notifications.user_id, notifications.announcement_id, expo_push_token as "to", announcements.title as title, announcements.message as body
FROM notifications
JOIN users ON users.id = notifications.user_id
JOIN announcements on announcements.id = notifications.announcement_id
WHERE notifications.status = "queued"
LIMIT 100
FOR UPDATE SKIP LOCKED
`

	var notifications []ExpoNotification
	if err := tx.Select(&notifications, query); err != nil {
		tx.Rollback()
		panic(err)
	}

	if len(notifications) == 0 {
		log.Println("Found no queued notifications to send.")
		tx.Rollback()
		return
	}

	// send to expo api
	tickets := SendNotifications(notifications)

	// update 100 w/ ticket numbers
	var updatedNotifications []model.Notification
	for i, notification := range notifications {
		updatedNotifications = append(updatedNotifications, model.Notification{
			UserID:         notification.UserID,
			AnnouncementID: notification.AnnouncementID,
			Status:         tickets.Data[i].Status,
			Receipt:        tickets.Data[i].ID,
		})
	}

	// Unfortunately there doesn't seem to be a good way to do this in one batch.
	for _, n := range updatedNotifications {
		query := `UPDATE notifications
	   SET status = :status, receipt = :receipt
	   WHERE user_id = :user_id and announcement_id = :announcement_id`
		_, err := tx.NamedExec(query, n)
		if err != nil {
			panic(err)
		}
	}

	err := tx.Commit()
	if err != nil {
		tx.Rollback()
		panic(err)
	}
	log.Println("Finished SendNotifications worker.")
}
