package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

type Conference struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	StartDate string `db:"start_date"`
	EndDate   string `db:"end_date"`
}

type ConferenceOptions struct {
	ConvertTimeToUSPacific bool
}

func ListConferences(db *sqlx.DB, options ConferenceOptions) ([]Conference, error) {

	startTimeQuery := "start_date"
	endTimeQuery := "end_date"
	if options.ConvertTimeToUSPacific {
		startTimeQuery = `DATE_FORMAT(CONVERT_TZ(start_date, 'UTC','US/Pacific'), "%a, %b %e, %Y at %l:%i %p") as start_date`
		endTimeQuery = `DATE_FORMAT(CONVERT_TZ(end_date, 'UTC','US/Pacific'), "%a, %b %e, %Y at %l:%i %p") as end_date`
	}

	query := `SELECT id, name,` + startTimeQuery + `,` + endTimeQuery + ` FROM conferences`
	var conferences []Conference
	if err := db.Select(&conferences, query); err != nil {
		return conferences, fmt.Errorf("failed to list conferences: %w", err)
	}
	if conferences == nil {
		conferences = make([]Conference, 0)
	}
	return conferences, nil
}

func GetConferenceByID(db *sqlx.DB, id string) (Conference, error) {
	const query = `
SELECT id, name, start_date, end_date
FROM conferences
WHERE id = ?
`
	var conferences []Conference
	if err := db.Select(&conferences, query, id); err != nil {
		return Conference{}, fmt.Errorf("failed to select conference: %w", err)
	}
	if len(conferences) == 0 {
		return Conference{}, errors.New("found no conference with given id")
	}
	return conferences[0], nil
}

func SaveConference(db *sqlx.DB, conference Conference) error {
	if conference.ID == 0 {
		return insertConference(db, conference)
	}
	return updateConference(db, conference)
}

func insertConference(db *sqlx.DB, conference Conference) error {
	query := "INSERT INTO conferences (name, start_date, end_date) VALUES (:name, :start_date, :end_date)"
	if _, err := db.NamedExec(query, conference); err != nil {
		return fmt.Errorf("failed to insert conference: %w", err)
	}
	return nil
}

func updateConference(db *sqlx.DB, conference Conference) error {
	query := "UPDATE conferences SET name = :name, start_date = :start_date, end_date = :end_date WHERE id = :id"
	if _, err := db.NamedExec(query, conference); err != nil {
		return fmt.Errorf("failed to update conference: %w", err)
	}
	return nil
}

func DeleteConference(db *sqlx.DB, id string) error {
	if id == "" {
		return errors.New("conference id must be provided")
	}
	const query = "DELETE FROM conferences WHERE id = ?"
	res, err := db.Exec(query, id)
	if err != nil {
		if strings.Contains(err.Error(), "a foreign key constraint fails") {
			return fmt.Errorf("cannot delete conference without first deleting all conference data")
		}
		return fmt.Errorf("failed to delete conference: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("failed to delete conference: no rows affected")
	}
	return nil
}
