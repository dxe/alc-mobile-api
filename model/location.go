package model

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

type Location struct {
	ID      int     `db:"id" json:"id"`
	Name    string  `db:"name" json:"name"`
	PlaceID string  `db:"place_id" json:"place_id"`
	Address string  `db:"address" json:"address"`
	City    string  `db:"city" json:"city"`
	Lat     float64 `db:"lat" json:"lat"`
	Lng     float64 `db:"lng" json:"lng"`
}

func getAllLocations(db *sqlx.DB) ([]Location, error) {
	// TODO: Implement this function.
	return nil, errors.New("not yet implemented")
}
