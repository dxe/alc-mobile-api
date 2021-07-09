package model

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type Info struct {
	ID       int    `db:"id" json:"id"`
	Title    string `db:"title" json:"title"`
	Subtitle string `db:"subtitle" json:"subtitle"`
	Content  string `db:"content" json:"content"`
	Icon     string `db:"icon" json:"icon"`
}

func getAllInfo(db *sqlx.DB) ([]Info, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
