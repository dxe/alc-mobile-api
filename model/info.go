package model

import (
	"errors"
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
	var info []Info
	if err := db.Select(&info, query); err != nil {
		return info, fmt.Errorf("failed to list info: %w", err)
	}
	if info == nil {
		return nil, errors.New("no info found")
	}
	return info, nil
}
