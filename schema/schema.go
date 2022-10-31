package schema

import (
	"database/sql"
	"fmt"
)

type Schema struct {
	Db *sql.DB
}

func NewSchema(dsn string) (*Schema, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("couldn't open the database: %w", err)
	}
	return &Schema{db}, nil
}

func (s *Schema) ForEachChannel(f func(Channel)) error {
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
		f(c)
	}

	return nil
}
