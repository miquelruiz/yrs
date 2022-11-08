package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	_ "github.com/mattn/go-sqlite3"
	"github.com/miquelruiz/yrs/lib"
	"github.com/spf13/cobra"
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
			c, err := loadConfig(ConfigPath)
			if err != nil {
				return fmt.Errorf(
					"couldn't load the config file %s: %w",
					ConfigPath,
					err,
				)
			}

			db, err := lib.New(c.DatabaseDriver, c.DatabaseUrl)
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
		RunE: func(cmd *cobra.Command, args []string) error {
			yrs := cmd.Context().Value(AppKey).(*lib.Yrs)
			return yrs.Update()
		},
	}

	subscribeCmd = &cobra.Command{
		Use:   "subscribe <ID>",
		Short: "Subscribe to the given channel",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			yrs := cmd.Context().Value(AppKey).(*lib.Yrs)
			return yrs.Subscribe(args[0])
		},
	}

	listVideosCmd = &cobra.Command{
		Use:   "list-videos",
		Short: "List all the videos in the database",
		RunE: func(cmd *cobra.Command, args []string) error {
			yrs := cmd.Context().Value(AppKey).(*lib.Yrs)
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
					v.ChannelId,
				)
			}

			return nil
		},
	}

	listChannelsCmd = &cobra.Command{
		Use:   "list-channels",
		Short: "List all the subscribed channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			yrs := cmd.Context().Value(AppKey).(*lib.Yrs)
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
		},
	}
)

func main() {
	rootCmd.PersistentFlags().StringVarP(
		&ConfigPath,
		"config",
		"c",
		"",
		"Path to config file",
	)

	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(subscribeCmd)
	rootCmd.AddCommand(listVideosCmd)
	rootCmd.AddCommand(listChannelsCmd)

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		panic(err)
	}
}