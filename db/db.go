package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const dbPath = "fingerprints.sqlite"

var DB *sql.DB

func InitDB() error {
	var err error

	DB, err = sql.Open("sqlite3", getDBPath())
	if err != nil {
		return err
	}

	if _, err := DB.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return err
	}

	return createTables()
}

func CurrentDBPath() string {
	return getDBPath()
}

func getDBPath() string {
	if value := strings.TrimSpace(os.Getenv("TRACKSEEK_DB_PATH")); value != "" {
		return value
	}

	return dbPath
}

func Close() error {
	if DB == nil {
		return nil
	}

	return DB.Close()
}

func createTables() error {
	if _, err := DB.Exec(`CREATE TABLE IF NOT EXISTS artists (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);`); err != nil {
		return err
	}

	if err := ensureTracksTable(); err != nil {
		return err
	}

	return ensureFingerprintsTable()
}

func ensureTracksTable() error {
	rows, err := DB.Query(`PRAGMA table_info(tracks);`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasRows := false
	hasPath := false
	hasTitle := false
	hasArtistID := false

	for rows.Next() {
		hasRows = true

		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue any
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}

		if name == "path" && strings.EqualFold(dataType, "TEXT") {
			hasPath = true
		}

		if name == "title" && strings.EqualFold(dataType, "TEXT") {
			hasTitle = true
		}

		if name == "artist_id" && strings.EqualFold(dataType, "INTEGER") {
			hasArtistID = true
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	hasArtistForeignKey, err := tracksHaveArtistForeignKey()
	if err != nil {
		return err
	}

	if hasRows && hasPath && hasTitle && hasArtistID && hasArtistForeignKey {
		_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_tracks_artist_id ON tracks(artist_id);`)
		return err
	}

	if hasRows {
		return fmt.Errorf("tracks schema mismatch detected; refusing to modify existing tables automatically")
	}

	statements := []string{
		`CREATE TABLE tracks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT NOT NULL,
			title TEXT NOT NULL,
			artist_id INTEGER NOT NULL,
			FOREIGN KEY(artist_id) REFERENCES artists(id)
		);`,
		`CREATE INDEX idx_tracks_artist_id ON tracks(artist_id);`,
	}

	for _, statement := range statements {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("tracks schema creation failed: %w", err)
		}
	}

	return nil
}

func ensureFingerprintsTable() error {
	rows, err := DB.Query(`PRAGMA table_info(fingerprints);`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasRows := false
	hashIsInteger := false
	hasTrackID := false

	for rows.Next() {
		hasRows = true

		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue any
		var pk int

		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}

		if name == "hash" && strings.EqualFold(dataType, "INTEGER") {
			hashIsInteger = true
		}

		if name == "track_id" && strings.EqualFold(dataType, "INTEGER") {
			hasTrackID = true
		}
	}

	if err := rows.Err(); err != nil {
		return err
	}

	hasTrackForeignKey, err := fingerprintsHasTrackForeignKey()
	if err != nil {
		return err
	}

	if hasRows && hashIsInteger && hasTrackID && hasTrackForeignKey {
		_, err = DB.Exec(`CREATE INDEX IF NOT EXISTS idx_fingerprints_hash ON fingerprints(hash);`)
		return err
	}

	if hasRows {
		return fmt.Errorf("fingerprints schema mismatch detected; refusing to modify existing tables automatically")
	}

	statements := []string{
		`CREATE TABLE fingerprints (
			hash INTEGER NOT NULL,
			track_id INTEGER NOT NULL,
			time_ms INTEGER NOT NULL,
			FOREIGN KEY(track_id) REFERENCES tracks(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX idx_fingerprints_hash ON fingerprints(hash);`,
	}

	for _, statement := range statements {
		if _, err := DB.Exec(statement); err != nil {
			return fmt.Errorf("fingerprints schema creation failed: %w", err)
		}
	}

	return nil
}

func tracksHaveArtistForeignKey() (bool, error) {
	rows, err := DB.Query(`PRAGMA foreign_key_list(tracks);`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var seq int
		var tableName string
		var fromColumn string
		var toColumn string
		var onUpdate string
		var onDelete string
		var match string

		if err := rows.Scan(&id, &seq, &tableName, &fromColumn, &toColumn, &onUpdate, &onDelete, &match); err != nil {
			return false, err
		}

		if tableName == "artists" && fromColumn == "artist_id" && toColumn == "id" {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func fingerprintsHasTrackForeignKey() (bool, error) {
	rows, err := DB.Query(`PRAGMA foreign_key_list(fingerprints);`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var seq int
		var tableName string
		var fromColumn string
		var toColumn string
		var onUpdate string
		var onDelete string
		var match string

		if err := rows.Scan(&id, &seq, &tableName, &fromColumn, &toColumn, &onUpdate, &onDelete, &match); err != nil {
			return false, err
		}

		if tableName == "tracks" && fromColumn == "track_id" && toColumn == "id" {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}
