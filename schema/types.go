package schema

import (
	"time"
)

type Channel struct {
	ID           string
	URL          string
	Name         string
	RSS          string
	Autodownload bool
}

type Video struct {
	ID         string
	URL        string
	Title      string
	Published  time.Time
	Channel_id string
	Downloaded bool
}
