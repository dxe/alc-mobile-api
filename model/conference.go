package model

import (
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type Conference struct {
	ID        int       `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	StartDate time.Time `db:"start_date" json:"start_date"`
	EndDate   time.Time `db:"end_date" json:"end_date"`
}

func ListConferences(db *sqlx.DB) ([]Conference, error) {
	const query = "SELECT id, name, start_date, end_date FROM conferences"
	var conferences []Conference
	if err := db.Select(&conferences, query); err != nil {
		return conferences, fmt.Errorf("failed to list conferences: %w", err)
	}
	if conferences == nil {
		return nil, errors.New("no conferences found")
	}
	return conferences, nil
}

func (c *Conference) Create(db *sqlx.DB) (*Conference, error) {
	// TODO: Implement this function.
	return c, errors.New("not yet implemented")
}

func (c *Conference) GetByID(db *sqlx.DB, id int) (*Conference, error) {
	// TODO: Implement this function.
	return c, errors.New("not yet implemented")
}

func (c *Conference) Update(db *sqlx.DB, id int) (*Conference, error) {
	// TODO: Implement this function.
	return c, errors.New("not yet implemented")
}
