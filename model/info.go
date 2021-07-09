package model

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Info struct {
	ID       int    `db:"id" json:"id"`
	Title    string `db:"title" json:"title"`
	Subtitle string `db:"subtitle" json:"subtitle"`
	Content  string `db:"content" json:"content"`
	Icon     string `db:"icon" json:"icon"`
}

func ListInfo(db *sqlx.DB) ([]Info, error) {
	const query = "SELECT id, title, subtitle, content, icon FROM info"
	info := make([]Info, 0)
	if err := db.Select(&info, query); err != nil {
		return info, fmt.Errorf("failed to list info: %w", err)
	}
	return info, nil
}
