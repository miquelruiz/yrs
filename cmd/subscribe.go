package cmd

import (
	"fmt"

	"github.com/miquelruiz/youtube-rss-subscriber-go/schema"
	"github.com/mmcdole/gofeed"
	"github.com/spf13/cobra"
)

const (
	urlFormat = "https://www.youtube.com/channel/%s"
	rssFormat = "https://www.youtube.com/feeds/videos.xml?channel_id=%s"
)

func init() {
	rootCmd.AddCommand(subscribeCmd)
}

var subscribeCmd = &cobra.Command{
	Use:   "subscribe <ID>",
	Short: "Subscribe to the given channel",
	Args:  cobra.ExactArgs(1),
	RunE:  subscribe,
}

func subscribe(cmd *cobra.Command, args []string) error {
	parser := gofeed.NewParser()
	rss := fmt.Sprintf(rssFormat, args[0])
	feed, err := parser.ParseURL(rss)
	if err != nil {
		return err
	}

	channel := schema.Channel{
		ID:           args[0],
		URL:          fmt.Sprintf(urlFormat, args[0]),
		Name:         feed.Title,
		RSS:          rss,
		Autodownload: false,
	}

	db := cmd.Context().Value(DbKey).(*schema.Schema)
	if err := db.InsertChannel(channel); err != nil {
		return err
	}

	fmt.Printf("Subscribed to %s\n", channel.Name)
	updateChannelVideos(db, &channel)

	return nil
}
