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

func EnsureArtistTx(tx *sql.Tx, name string) (int64, error) {
	result, err := tx.Exec(`INSERT OR IGNORE INTO artists(name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}

	if id, err := result.LastInsertId(); err == nil && id > 0 {
		return id, nil
	}

	row := tx.QueryRow(`SELECT id FROM artists WHERE name = ?`, name)

	var artistID int64
	if err := row.Scan(&artistID); err != nil {
		return 0, err
	}

	return artistID, nil
}
