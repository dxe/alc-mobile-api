package model

type Image struct {
	ID  int    `db:"id" json:"id"`
	URL string `db:"url" json:"url"`
}

// TODO
