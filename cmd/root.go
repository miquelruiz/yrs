package cmd

import (
	"context"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/miquelruiz/youtube-rss-subscriber-go/schema"
	"github.com/spf13/cobra"
)

type KeyType int

const (
	DbKey KeyType = iota
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

			db, err := schema.NewSchema(c.DatabaseUrl)
			if err != nil {
				return fmt.Errorf("couldn't open the database: %w", err)
			}

			ctx := context.WithValue(cmd.Context(), DbKey, db)
			cmd.SetContext(ctx)

			return nil
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&ConfigPath,
		"config",
		"c",
		"",
		"Path to config file",
	)
}

func Execute() error {
	return rootCmd.ExecuteContext(context.Background())
}
