package yrs

import (
	"crypto/sha1"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/mmcdole/gofeed"
	"github.com/samber/lo"
)

const (
	rssFormat = "https://www.youtube.com/feeds/videos.xml?channel_id=%s"
)

type Yrs struct {
	db *sql.DB
}

func New(driver, dsn string) (*Yrs, error) {
	// Check if there's any schema migrations to run. That function is in
	// charge of creating the db if it doesn't exist.
	db, err := ManageSchema(driver, dsn)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("PRAGMA foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("couldn't enable foreign keys: %w", err)
	}

	return &Yrs{db}, err
}

func (y *Yrs) forEachChannel(f func(*Channel) error) error {
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
		err = f(&c)
		if err != nil {
			return fmt.Errorf("error in callback: %w", err)
		}
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

func (y *Yrs) insertChannel(tx *sql.Tx, c Channel) error {
	insert, err := tx.Prepare(`
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
	err := y.forEachChannel(func(c *Channel) error {
		rowSlice = append(rowSlice, *c)
		return nil
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

func (y *Yrs) SubscribeYouTubeID(channelStr string) error {
	return y.Subscribe(fmt.Sprintf(rssFormat, channelStr))
}

func (y *Yrs) Subscribe(rss string) error {
	feed, err := gofeed.NewParser().ParseURL(rss)
	if err != nil {
		return fmt.Errorf("error parsing RSS url %s: %w", rss, err)
	}

	s := sha1.New()
	s.Write([]byte(rss))

	return y.subscribeChannel(Channel{
		ID:           fmt.Sprintf("%x", s.Sum(nil))[:24],
		URL:          feed.Link,
		Name:         feed.Title,
		RSS:          rss,
		Autodownload: false,
	}, feed)
}

func (y *Yrs) subscribeChannel(channel Channel, feed *gofeed.Feed) error {
	tx, err := y.db.Begin()
	if err != nil {
		return fmt.Errorf("error on begin: %w", err)
	}

	if err := y.insertChannel(tx, channel); err != nil {
		tx.Rollback()
		return fmt.Errorf("error inserting channel: %w", err)
	}

	err = updateChannelVideos(tx, &channel, nil, feed)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("error updating channel videos: %w", err)
	}

	return tx.Commit()
}

func (y *Yrs) Update() ([]Video, error) {
	errors_ := make(chan error, 100)
	videos := make(chan Video, 100)
	retErrors := make([]error, 0)
	retVideos := make([]Video, 0)

	tx, err := y.db.Begin()
	if err != nil {
		return retVideos, err
	}

	var wg sync.WaitGroup
	err = y.forEachChannel(func(c *Channel) error {
		wg.Add(1)
		go func() {
			defer wg.Done()
			feed, e := gofeed.NewParser().ParseURL(c.RSS)
			if e != nil {
				errors_ <- fmt.Errorf("error retrieving %s: %s", c.RSS, e)
				return
			}
			e = updateChannelVideos(tx, c, videos, feed)
			if e != nil {
				errors_ <- e
			}
		}()
		return nil
	})

	if err != nil {
		tx.Rollback()
		return retVideos, err
	}

	go func() {
		for e := range errors_ {
			retErrors = append(retErrors, e)
		}
	}()

	go func() {
		for v := range videos {
			retVideos = append(retVideos, v)
		}
	}()

	wg.Wait()
	close(videos)
	close(errors_)

	if len(retErrors) > 0 {
		tx.Rollback()
		return retVideos, errors.Join(retErrors...)
	}

	err = tx.Commit()
	return retVideos, err
}

func updateChannelVideos(tx *sql.Tx, c *Channel, vc chan Video, feed *gofeed.Feed) error {
	insert, err := tx.Prepare(
		`INSERT INTO videos (id, url, title, published, channel_id, downloaded)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}

	ftsInsert, err := tx.Prepare(
		`INSERT INTO videos_fts (id, title, channel) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return err
	}

	for _, item := range feed.Items {
		date, err := parseDate(item.Published)
		if err != nil {
			return fmt.Errorf(
				"error parsing date (%s) for video %s: %w",
				item.Published,
				item.Title,
				err,
			)
		}
		v := Video{
			ID:         getVideoID(item),
			URL:        item.Link,
			Title:      item.Title,
			Published:  date,
			ChannelId:  c.ID,
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

		_, err = ftsInsert.Exec(v.ID, v.Title, c.Name)
		if err != nil {
			return err
		}

		if vc != nil {
			vc <- v
		}
	}

	return nil
}

func (y *Yrs) Unsubscribe(channelID string) error {
	delete, err := y.db.Prepare("DELETE FROM channels WHERE id=?")
	if err != nil {
		return err
	}

	_, err = delete.Exec(channelID)
	return err
}

func (y *Yrs) Search(s string) ([]SearchResult, error) {
	rows, err := y.db.Query(
		"SELECT * FROM videos_fts WHERE videos_fts MATCH ? ORDER BY rank",
		s,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]SearchResult, 0)
	for rows.Next() {
		res := SearchResult{}
		err = rows.Scan(&res.ID, &res.Title, &res.Channel)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		results = append(results, res)
	}

	return results, nil
}

func (y *Yrs) GetVideosByID(ids []string) ([]Video, error) {
	query := `
		SELECT
			v.id, v.title, v.url, v.published, v.channel_id, v.downloaded,
			c.id, c.url, c.name, c.rss, c.autodownload
		FROM videos v
		JOIN channels c ON (v.channel_id=c.id)
		WHERE v.id IN (%s)
	`
	placeholders := strings.Join(strings.Split(strings.Repeat("?", len(ids)), ""), ",")

	s := lo.Map(ids, func(item string, index int) interface{} {
		return interface{}(&item)
	})
	rows, err := y.db.Query(fmt.Sprintf(query, placeholders), s...)
	if err != nil {
		return nil, err
	}

	videos := make([]Video, 0)
	for rows.Next() {
		v := Video{}
		c := Channel{}
		err = rows.Scan(
			&v.ID, &v.Title, &v.URL, &v.Published, &v.ChannelId, &v.Downloaded,
			&c.ID, &c.URL, &c.Name, &c.RSS, &c.Autodownload,
		)
		if err != nil {
			return nil, err
		}
		v.Channel = &c
		videos = append(videos, v)
	}

	return videos, nil
}

func (y *Yrs) GetVideosByChannel(ch string) ([]Video, error) {
	query := `
		SELECT
			v.id, v.title, v.url, v.published, v.channel_id, v.downloaded,
			c.id, c.url, c.name, c.rss, c.autodownload
		FROM videos v
		JOIN channels c ON (v.channel_id=c.id)
		WHERE c.name=?
		ORDER BY v.published
	`
	log.Printf("Listing videos for channel %s", ch)
	rows, err := y.db.Query(query, ch)
	if err != nil {
		return nil, err
	}

	videos := make([]Video, 0)
	for rows.Next() {
		v := Video{}
		c := Channel{}
		err := rows.Scan(
			&v.ID, &v.Title, &v.URL, &v.Published, &v.ChannelId, &v.Downloaded,
			&c.ID, &c.URL, &c.Name, &c.RSS, &c.Autodownload,
		)
		if err != nil {
			return nil, err
		}
		v.Channel = &c
		videos = append(videos, v)
	}

	return videos, nil
}

func (y *Yrs) DeleteChannel(ch string) error {
	query := "DELETE FROM channels WHERE id=?"
	_, err := y.db.Exec(query, ch)
	return err
}

func parseDate(dateStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		time.RFC850,
		time.RFC1123Z,
	}

	var date time.Time
	var err error
	for _, format := range formats {
		date, err = time.Parse(format, dateStr)
		if err == nil {
			break
		}
	}
	return date, err
}

func getVideoID(item *gofeed.Item) string {
	s := sha1.New()
	s.Write([]byte(item.Link))
	return fmt.Sprintf("%x", s.Sum(nil))[:10]
}
