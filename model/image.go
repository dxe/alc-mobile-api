package model

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type Image struct {
	ID  int    `db:"id" json:"id"`
	URL string `db:"url" json:"url"`
}

func ListImages(db *sqlx.DB) ([]Image, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
