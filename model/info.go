package model

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Info struct {
	ID           int    `db:"id"`
	Title        string `db:"title"`
	Subtitle     string `db:"subtitle"`
	Content      string `db:"content"`
	Icon         string `db:"icon"`
	DisplayOrder int    `db:"display_order"`
}

func ListInfo(db *sqlx.DB) ([]Info, error) {
	const query = "SELECT id, title, subtitle, content, icon, display_order FROM info ORDER BY display_order"
	var info []Info
	if err := db.Select(&info, query); err != nil {
		return info, fmt.Errorf("failed to list info: %w", err)
	}
	if info == nil {
		info = make([]Info, 0)
	}
	return info, nil
}

func GetInfoByID(db *sqlx.DB, id string) (Info, error) {
	const query = `
SELECT id, title, subtitle, content, icon, display_order
FROM info
WHERE id = ?
`
	var info []Info
	if err := db.Select(&info, query, id); err != nil {
		return Info{}, fmt.Errorf("failed to select info: %w", err)
	}
	if len(info) == 0 {
		return Info{}, errors.New("found no info with given id")
	}
	return info[0], nil
}

func SaveInfo(db *sqlx.DB, info Info) error {
	if info.ID == 0 {
		return insertInfo(db, info)
	}
	return updateInfo(db, info)
}

func insertInfo(db *sqlx.DB, info Info) error {
	query := `
INSERT INTO info (title, subtitle, content, icon, display_order)
VALUES (:title, :subtitle, :content, :icon, :display_order)
`
	if _, err := db.NamedExec(query, info); err != nil {
		return fmt.Errorf("failed to insert info: %w", err)
	}
	return nil
}

func updateInfo(db *sqlx.DB, info Info) error {
	query := `
UPDATE info
SET title = :title, subtitle = :subtitle, content = :content, icon = :icon, display_order = :display_order
WHERE id = :id
`
	if _, err := db.NamedExec(query, info); err != nil {
		return fmt.Errorf("failed to update info: %w", err)
	}
	return nil
}

func DeleteInfo(db *sqlx.DB, id string) error {
	if id == "" {
		return errors.New("info id must be provided")
	}
	const query = "DELETE FROM info WHERE id = ?"
	res, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete info: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("failed to delete info: no rows affected")
	}
	return nil
}
