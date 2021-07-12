package model

import (
	"errors"
	"fmt"
	"strings"

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
		locations = make([]Location, 0)
	}
	return locations, nil
}

func GetLocationByID(db *sqlx.DB, id string) (Location, error) {
	const query = `
SELECT id, name, place_id, address, city, lat, lng
FROM locations
WHERE id = ?
`
	var locations []Location
	if err := db.Select(&locations, query, id); err != nil {
		return Location{}, fmt.Errorf("failed to select location: %w", err)
	}
	if len(locations) == 0 {
		return Location{}, errors.New("found no location with given id")
	}
	return locations[0], nil
}

func SaveLocation(db *sqlx.DB, location Location) error {
	if location.ID == 0 {
		return insertLocation(db, location)
	}
	return updateLocation(db, location)
}

func insertLocation(db *sqlx.DB, location Location) error {
	query := `
INSERT INTO locations (name, place_id, address, city, lat, lng)
VALUES (:name, :place_id, :address, :city, :lat, :lng)
`
	if _, err := db.NamedExec(query, location); err != nil {
		return fmt.Errorf("failed to insert location: %w", err)
	}
	return nil
}

func updateLocation(db *sqlx.DB, location Location) error {
	query := `
UPDATE locations
SET name = :name, place_id = :place_id, address = :address, city = :city, lat = :lat, lng = :lng
WHERE id = :id
`
	if _, err := db.NamedExec(query, location); err != nil {
		return fmt.Errorf("failed to update location: %w", err)
	}
	return nil
}

func DeleteLocation(db *sqlx.DB, id string) error {
	if id == "" {
		return errors.New("location id must be provided")
	}
	const query = "DELETE FROM locations WHERE id = ?"
	res, err := db.Exec(query, id)
	if err != nil {
		if strings.Contains(err.Error(), "a foreign key constraint fails") {
			return fmt.Errorf("cannot delete location because it is used by events")
		}
		return fmt.Errorf("failed to delete location: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("failed to delete location: no rows affected")
	}
	return nil
}
