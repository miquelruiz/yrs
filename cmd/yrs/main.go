package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/tabwriter"

	"github.com/miquelruiz/yrs/internal/config"
	"github.com/miquelruiz/yrs/internal/vcs"
	"github.com/miquelruiz/yrs/pkg/yrs"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type KeyType int

const (
	AppKey KeyType = iota
)

var (
	ConfigPath string
	rootCmd    = &cobra.Command{
		Use:   "yrs",
		Short: "YouTube RSS Subscriber",
		Long:  "A tool to subscribe to YouTube channels without a YouTube account",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			c, err := config.Load(ConfigPath)
			if err != nil {
				return fmt.Errorf(
					"couldn't load the config file %s: %w",
					ConfigPath,
					err,
				)
			}

			db, err := yrs.New(c.DatabaseDriver, c.DatabaseUrl)
			if err != nil {
				return fmt.Errorf("couldn't create schema: %w", err)
			}

			ctx := context.WithValue(cmd.Context(), AppKey, db)
			cmd.SetContext(ctx)

			return nil
		},
	}

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update subscriptions",
		RunE:  update,
	}

	subscribeYouTubeCmd = &cobra.Command{
		Use:   "subscribe-yt <YouTube channel URL>",
		Short: "Subscribe to the given channel",
		Args:  cobra.ExactArgs(1),
		RunE:  subscribeYouTube,
	}

	subscribeCmd = &cobra.Command{
		Use:   "subscribe <rss url>",
		Short: "Subscribe to the given feed",
		Args:  cobra.ExactArgs(1),
		RunE:  subscribe,
	}

	listVideosCmd = &cobra.Command{
		Use:   "list-videos",
		Short: "List all the videos in the database",
		RunE:  listVideos,
	}

	listChannelsCmd = &cobra.Command{
		Use:   "list-channels",
		Short: "List all the subscribed channels",
		RunE:  listChannels,
	}

	unsubscribeCmd = &cobra.Command{
		Use:   "unsubscribe <Channel ID>",
		Short: "Unsubscribe from the given channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
			return yrs.Unsubscribe(args[0])
		},
	}

	searchCmd = &cobra.Command{
		Use:   "search <search term>",
		Short: "Search video titles and channels",
		Args:  cobra.ExactArgs(1),
		RunE:  search,
	}

	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version to stdout",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Printf("Version %s\nBuilt on %s\n", vcs.Version, vcs.Time)
		},
	}
)

func getChannelID(r io.Reader) string {
	channelID := ""
	h := html.NewTokenizer(r)

LOOP:
	for {
		tt := h.Next()
		switch {
		case tt == html.ErrorToken:
			break LOOP
		default:
			t := h.Token()
			if t.DataAtom != atom.Meta {
				continue
			}

			attrMap := map[string]string{}
			for _, a := range t.Attr {
				attrMap[a.Key] = a.Val
			}

			if attrMap["itemprop"] == "identifier" ||
				attrMap["itemprop"] == "channelId" {
				channelID = attrMap["content"]
				break LOOP
			}
		}
	}

	return channelID
}

func subscribeYouTube(cmd *cobra.Command, args []string) error {
	res, err := http.Get(args[0])
	if err != nil {
		return err
	}
	defer res.Body.Close()

	channelID := getChannelID(res.Body)
	if channelID == "" {
		return fmt.Errorf("channelID not found in %s", args[0])
	}

	yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
	return yrs.SubscribeYouTubeID(channelID)
}

func subscribe(cmd *cobra.Command, args []string) error {
	yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
	return yrs.Subscribe(args[0])
}

func update(cmd *cobra.Command, args []string) error {
	yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
	videos, err := yrs.Update()
	if err != nil {
		return err
	}

	for _, v := range videos {
		fmt.Printf(
			"Title: %s\nChannel: %s\nURL: %s\n\n",
			v.Title,
			v.Channel.Name,
			v.URL,
		)
	}

	return nil
}

func search(cmd *cobra.Command, args []string) error {
	yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
	results, err := yrs.Search(args[0])
	if err != nil {
		return err
	}

	if len(results) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tTitle\tChannel\tURL")

	for _, r := range results {
		url := fmt.Sprintf("https://www.youtube.com/watch?v=%s", r.ID)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.ID, r.Title, r.Channel, url)
	}

	return nil
}

func listVideos(cmd *cobra.Command, args []string) error {
	yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
	videos, err := yrs.GetVideos()
	if err != nil {
		return err
	}
	if len(videos) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "ID\tTitle\tURL\tPublished\tChannelId")

	for _, v := range videos {
		fmt.Fprintf(
			w,
			"%s\t%s\t%s\t%s\t%s\t\n",
			v.ID,
			v.Title,
			v.URL,
			v.Published,
			v.Channel.Name,
		)
	}

	return nil
}

func listChannels(cmd *cobra.Command, args []string) error {
	yrs := cmd.Context().Value(AppKey).(*yrs.Yrs)
	channels, err := yrs.GetChannels()
	if err != nil {
		return err
	}
	if len(channels) == 0 {
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', 0)
	defer w.Flush()
	fmt.Fprintln(w, "#\tID\tName\tURL\tAutodownload")

	for i, c := range channels {
		fmt.Fprintf(
			w,
			"%d\t%s\t%s\t%s\t%t\t\n",
			i,
			c.ID,
			c.Name,
			c.URL,
			c.Autodownload,
		)
	}

	return nil
}

func main() {
	rootCmd.PersistentFlags().StringVarP(
		&ConfigPath,
		"config",
		"c",
		"",
		"Path to config file",
	)

	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(subscribeYouTubeCmd)
	rootCmd.AddCommand(subscribeCmd)
	rootCmd.AddCommand(listVideosCmd)
	rootCmd.AddCommand(listChannelsCmd)
	rootCmd.AddCommand(unsubscribeCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		panic(err)
	}
}
