package cmd

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mattn/go-sqlite3"
	"github.com/miquelruiz/youtube-rss-subscriber-go/schema"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(updateCmd)
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update subscriptions",
	RunE:  update,
}

func update(cmd *cobra.Command, args []string) error {
	db := cmd.Context().Value(DbKey).(*schema.Schema)

	insert, err := db.Db.Prepare(
		`INSERT INTO videos (id, url, title, published, channel_id, downloaded)
		VALUES (?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	err = db.ForEachChannel(func(c schema.Channel) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			parser := gofeed.NewParser()
			feed, err := parser.ParseURL(c.RSS)
			if err != nil {
				fmt.Printf("error retrieving %s: %v\n", c.RSS, err)
			}
			for _, item := range feed.Items {
				date, err := time.Parse(time.RFC3339, item.Published)
				if err != nil {
					fmt.Printf(
						"error parsing date (%s) for video %s: %v\n",
						item.Published,
						item.Title,
						err,
					)
				}
				_, err = insert.Exec(
					item.Extensions["yt"]["videoId"][0].Value,
					item.Link,
					item.Title,
					date,
					item.Extensions["yt"]["channelId"][0].Value,
					0,
				)

				var sqliteErr sqlite3.Error
				if errors.As(err, &sqliteErr) {
					if !errors.Is(sqliteErr.ExtendedCode, sqlite3.ErrConstraintPrimaryKey) {
						fmt.Println(err)
					}
					continue
				}

				fmt.Printf(
					"Channel: %s\nTitle: %s\nURL: %s\n\n",
					c.Name,
					item.Title,
					item.Link,
				)
			}
		}()
	})

	if err != nil {
		return err
	}

	wg.Wait()
	return nil
}