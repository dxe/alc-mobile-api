package model

import (
	"log"

	"github.com/jmoiron/sqlx"
)

func NewDB(connectionString string) *sqlx.DB {
	db, err := sqlx.Open("mysql", connectionString)
	if err != nil {
		panic(err)
	}
	log.Println("Connected to database.")
	// TODO(jhobbs): Init db here or in main?
	InitDatabase(db)
	return db
}

// WipeDatabase drops all tables in the database.
// It may be used for testing purposes, but is not planned to be used
// on the actual dev or prod database.
func WipeDatabase(db *sqlx.DB) {
	// TODO(jhobbs): Return an error if prod.
	db.MustExec(`DROP TABLE IF EXISTS conferences`)
	db.MustExec(`DROP TABLE IF EXISTS users`)
	db.MustExec(`DROP TABLE IF EXISTS events`)
	db.MustExec(`DROP TABLE IF EXISTS locations`)
	db.MustExec(`DROP TABLE IF EXISTS rsvp`)
	db.MustExec(`DROP TABLE IF EXISTS images`)
	db.MustExec(`DROP TABLE IF EXISTS info`)
	db.MustExec(`DROP TABLE IF EXISTS announcements`)
}

func InitDatabase(db *sqlx.DB) {
	db.MustExec(`
CREATE TABLE IF NOT EXISTS conferences (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(80) NOT NULL,
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(200) NOT NULL DEFAULT '',
    email VARCHAR(200) NOT NULL DEFAULT '',
    device_id VARCHAR(200),
    device_name VARCHAR(200),
    platform VARCHAR(60)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS events (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(200),
    description TEXT,
    start_time DATETIME NOT NULL,
    length FLOAT(3,2) NOT NULL,
    location_id INTEGER,
    image_id INTEGER,
    key_event TINYINT NOT NULL DEFAULT '0'
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS locations (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	name VARCHAR(200) NOT NULL,
    place_id VARCHAR(200),
    address VARCHAR(200) NOT NULL,
    city VARCHAR(100) NOT NULL,
    lat FLOAT(10,6),
    lng FLOAT(10,6)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS rsvp (
	event_id INTEGER NOT NULL,
	user_id INTEGER NOT NULL,
	attending TINYINT NOT NULL DEFAULT '0',
    PRIMARY KEY (event_id, user_id)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS images (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	url VARCHAR(400) NOT NULL
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS info (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	title VARCHAR(200) NOT NULL,
    subtitle VARCHAR(200) NOT NULL,
    content TEXT,
    icon VARCHAR(30)
)
`)

	db.MustExec(`
CREATE TABLE IF NOT EXISTS announcements (
	id INTEGER PRIMARY KEY AUTO_INCREMENT,
	title VARCHAR(200) NOT NULL,
    message TEXT NOT NULL,
    icon VARCHAR(30) NOT NULL,
    created_by VARCHAR(100) NOT NULL,
    send_time DATETIME,
    sent TINYINT NOT NULL DEFAULT '0'
)
`)

}
