package models

import (
	"database/sql"

	"trackseek/db"
)

type Track struct {
	ID       int64
	Path     string
	Title    string
	ArtistID int64
	Artist   *Artist
}

func (t Track) Save() (int64, error) {
	artistID, err := EnsureArtist(t.Artist.Name)
	if err != nil {
		return 0, err
	}

	result, err := db.DB.Exec(`INSERT INTO tracks(path, title, artist_id) VALUES (?, ?, ?)`, t.Path, t.Title, artistID)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (t Track) UpsertByPathTx(tx *sql.Tx) (int64, error) {
	artistID, err := EnsureArtistTx(tx, t.Artist.Name)
	if err != nil {
		return 0, err
	}

	row := tx.QueryRow(`SELECT id FROM tracks WHERE path = ?`, t.Path)

	var trackID int64
	if err := row.Scan(&trackID); err != nil {
		if err == sql.ErrNoRows {
			result, err := tx.Exec(`INSERT INTO tracks(path, title, artist_id) VALUES (?, ?, ?)`, t.Path, t.Title, artistID)
			if err != nil {
				return 0, err
			}

			return result.LastInsertId()
		}

		return 0, err
	}

	if _, err := tx.Exec(`UPDATE tracks SET title = ?, artist_id = ? WHERE id = ?`, t.Title, artistID, trackID); err != nil {
		return 0, err
	}

	return trackID, nil
}

func GetTrackByID(id int64) (*Track, error) {
	row := db.DB.QueryRow(`
		SELECT tracks.id, tracks.path, tracks.title, tracks.artist_id, artists.name
		FROM tracks
		JOIN artists ON artists.id = tracks.artist_id
		WHERE tracks.id = ?
	`, id)

	var track Track
	var artistName string
	if err := row.Scan(&track.ID, &track.Path, &track.Title, &track.ArtistID, &artistName); err != nil {
		return nil, err
	}

	track.Artist = &Artist{ID: track.ArtistID, Name: artistName}

	return &track, nil
}

func ListTracks() ([]Track, error) {
	rows, err := db.DB.Query(`
		SELECT tracks.id, tracks.path, tracks.title, tracks.artist_id, artists.name
		FROM tracks
		JOIN artists ON artists.id = tracks.artist_id
		ORDER BY artists.name ASC, tracks.title ASC, tracks.path ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var track Track
		var artistName string

		if err := rows.Scan(&track.ID, &track.Path, &track.Title, &track.ArtistID, &artistName); err != nil {
			return nil, err
		}

		track.Artist = &Artist{ID: track.ArtistID, Name: artistName}
		tracks = append(tracks, track)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tracks, nil
}
