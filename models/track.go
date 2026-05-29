package models

import "trackseek/db"

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
