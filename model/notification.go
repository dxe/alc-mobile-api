package model

type Notification struct {
	UserID         int    `db:"user_id"`
	AnnouncementID int    `db:"announcement_id"`
	Status         string `db:"status"`
	Receipt        string `db:"receipt"`
	ReceiptStatus  string `db:"receipt_status"`
	Timestamp      string `db:"timestamp"`
}

// TODO
