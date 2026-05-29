package models

import (
	"database/sql"
	"trackseek/db"
)

type Artist struct {
	ID   int64
	Name string
}

func EnsureArtist(name string) (int64, error) {
	result, err := db.DB.Exec(`INSERT OR IGNORE INTO artists(name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}

	if id, err := result.LastInsertId(); err == nil && id > 0 {
		return id, nil
	}

	row := db.DB.QueryRow(`SELECT id FROM artists WHERE name = ?`, name)

	var artistID int64
	if err := row.Scan(&artistID); err != nil {
		return 0, err
	}

	return artistID, nil
}

func GetArtistByID(id int64) (*Artist, error) {
	row := db.DB.QueryRow(`SELECT id, name FROM artists WHERE id = ?`, id)

	var artist Artist
	if err := row.Scan(&artist.ID, &artist.Name); err != nil {
		return nil, err
	}

	return &artist, nil
}

func scanArtist(row *sql.Row) (*Artist, error) {
	var artist Artist
	if err := row.Scan(&artist.ID, &artist.Name); err != nil {
		return nil, err
	}

	return &artist, nil
}
