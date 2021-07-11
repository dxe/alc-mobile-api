package model

import (
	"errors"
	"fmt"

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

func ListLocations(db *sqlx.DB) ([]Location, error) {
	const query = `
SELECT id, name, place_id, address, city, lat, lng FROM locations
ORDER BY name asc
`
	var locations []Location
	if err := db.Select(&locations, query); err != nil {
		return locations, fmt.Errorf("failed to list locations: %w", err)
	}
	if locations == nil {
		return nil, errors.New("no locations found")
	}
	return locations, nil
}
