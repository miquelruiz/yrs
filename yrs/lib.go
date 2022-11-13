package yrs

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
)

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

	urlFormat = "https://www.youtube.com/channel/%s"
	rssFormat = "https://www.youtube.com/feeds/videos.xml?channel_id=%s"
)

type Yrs struct {
	db *sql.DB
}

func New(driver, dsn string) (*Yrs, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("couldn't open the database: %w", err)
	}
	dbPath := strings.TrimPrefix(dsn, "file:")
	fileinfo, err := os.Stat(dbPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("couldn't stat the db '%s': %w", dbPath, err)
		}
		err = initializeDb(driver, dsn)
	} else if fileinfo.Size() == 0 {
		err = initializeDb(driver, dsn)
	}

	return &Yrs{db}, err
}

func (y *Yrs) forEachChannel(f func(*Channel)) error {
	rows, err := y.db.Query("SELECT * FROM channels")
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

func (y *Yrs) forEachVideo(f func(*Video)) error {
	rows, err := y.db.Query(`
		SELECT
			v.id, v.title, v.url, v.published, v.channel_id, v.downloaded,
			c.id, c.url, c.name, c.rss, c.autodownload
		FROM videos v
		JOIN channels c
		ON (v.channel_id = c.id)
		ORDER BY v.published
	`)
	if err != nil {
		return fmt.Errorf("couldn't retrieve the videos: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		v := Video{}
		c := Channel{}
		err = rows.Scan(
			&v.ID, &v.Title, &v.URL, &v.Published, &v.ChannelId, &v.Downloaded,
			&c.ID, &c.URL, &c.Name, &c.RSS, &c.Autodownload,
		)
		v.Channel = &c
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}
		f(&v)
	}

	return nil
}

func initializeDb(driver, dsn string) error {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	_, err = db.Exec(initStmt)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}
	return nil
}

func (y *Yrs) insertChannel(c Channel) error {
	insert, err := y.db.Prepare(`
		INSERT INTO channels (id, url, name, rss, autodownload)
		VALUES (?, ?, ?, ?, 0)
	`)
	if err != nil {
		return err
	}

	_, err = insert.Exec(c.ID, c.URL, c.Name, c.RSS)
	return err
}

func (y *Yrs) GetChannels() ([]Channel, error) {
	rowSlice := make([]Channel, 0)
	err := y.forEachChannel(func(c *Channel) {
		rowSlice = append(rowSlice, *c)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list channels: %w", err)
	}

	return rowSlice, nil
}

func (y *Yrs) GetVideos() ([]Video, error) {
	rowSlice := make([]Video, 0)
	err := y.forEachVideo(func(v *Video) {
		rowSlice = append(rowSlice, *v)
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list videos: %w", err)
	}

	return rowSlice, err
}

func (y *Yrs) Subscribe(channelStr string) error {
	parser := gofeed.NewParser()
	rss := fmt.Sprintf(rssFormat, channelStr)
	feed, err := parser.ParseURL(rss)
	if err != nil {
		return fmt.Errorf("error parsing RSS url %s: %w", rss, err)
	}

	channel := Channel{
		ID:           channelStr,
		URL:          fmt.Sprintf(urlFormat, channelStr),
		Name:         feed.Title,
		RSS:          rss,
		Autodownload: false,
	}

	if err := y.insertChannel(channel); err != nil {
		return err
	}

	y.updateChannelVideos(&channel, nil)
	return nil
}

func (y *Yrs) Update() ([]Video, error) {
	var wg sync.WaitGroup
	videos := make(chan Video, 100)
	err := y.forEachChannel(func(c *Channel) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			y.updateChannelVideos(c, videos)
		}()
	})

	retVideos := make([]Video, 0)
	if err != nil {
		return retVideos, err
	}

	go func() {
		for v := range videos {
			retVideos = append(retVideos, v)
		}
	}()

	wg.Wait()
	return retVideos, nil
}

func (y *Yrs) updateChannelVideos(c *Channel, vc chan Video) error {
	parser := gofeed.NewParser()
	feed, err := parser.ParseURL(c.RSS)
	if err != nil {
		return fmt.Errorf("error retrieving %s: %w", c.RSS, err)
	}

	insert, err := y.db.Prepare(
		`INSERT INTO videos (id, url, title, published, channel_id, downloaded)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}

	for _, item := range feed.Items {
		date, err := time.Parse(time.RFC3339, item.Published)
		if err != nil {
			return fmt.Errorf(
				"error parsing date (%s) for video %s: %w",
				item.Published,
				item.Title,
				err,
			)
		}
		v := Video{
			ID:         item.Extensions["yt"]["videoId"][0].Value,
			URL:        item.Link,
			Title:      item.Title,
			Published:  date,
			ChannelId:  item.Extensions["yt"]["channelId"][0].Value,
			Downloaded: false,
			Channel:    c,
		}
		_, err = insert.Exec(
			v.ID, v.URL, v.Title, v.Published, v.ChannelId, 0,
		)

		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) {
			if !errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
				return err
			}
			continue
		}

		if vc != nil {
			vc <- v
		}
	}

	return nil
}
