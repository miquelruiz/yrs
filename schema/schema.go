package schema

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

type Schema struct {
	Db *sql.DB
}

const (
	initStmt = `
CREATE TABLE channels (
	id VARCHAR(64) NOT NULL,
	url VARCHAR(256) NOT NULL,
	name VARCHAR(64) NOT NULL,
	rss VARCHAR(256) NOT NULL,
	autodownload INTEGER NOT NULL,
	PRIMARY KEY (id)
);
CREATE TABLE videos (
	id VARCHAR(64) NOT NULL,
	url VARCHAR(256) NOT NULL,
	title VARCHAR(256) NOT NULL,
	published DATETIME NOT NULL,
	channel_id INTEGER NOT NULL,
	downloaded INTEGER NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(channel_id) REFERENCES channels (id)
);`
)

func NewSchema(dsn string) (*Schema, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("couldn't open the database: %w", err)
	}
	dbPath := strings.TrimPrefix(dsn, "file:")
	fileinfo, err := os.Stat(dbPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("couldn't stat the db '%s': %w", dbPath, err)
		}
		InitializeDb(dsn)
	} else if fileinfo.Size() == 0 {
		InitializeDb(dsn)
	}

	return &Schema{db}, nil
}

func (s *Schema) ForEachChannel(f func(*Channel)) error {
	rows, err := s.Db.Query("SELECT * FROM channels")
	if err != nil {
		return fmt.Errorf("couldn't retrieve the channels: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		c := Channel{}
		err = rows.Scan(&c.ID, &c.URL, &c.Name, &c.RSS, &c.Autodownload)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		f(&c)
	}

	return nil
}

func InitializeDb(dsn string) error {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	_, err = db.Exec(initStmt)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func (s *Schema) InsertChannel(c Channel) error {
	insert, err := s.Db.Prepare(`
		INSERT INTO channels (id, url, name, rss, autodownload)
		VALUES (?, ?, ?, ?, 0)
	`)
	if err != nil {
		return err
	}

	_, err = insert.Exec(c.ID, c.URL, c.Name, c.RSS)
	return err
}
