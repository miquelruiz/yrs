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
	rootCmd = &cobra.Command{
		Use:   "yrs",
		Short: "YouTube RSS Subscriber",
		Long:  "A tool to subscribe to YouTube channels without a YouTube account",
	}
)

func Execute() error {
	c, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := schema.NewSchema(c.DatabaseUrl)
	if err != nil {
		return fmt.Errorf("couldn't open the database: %w", err)
	}

	ctx := context.WithValue(context.Background(), DbKey, db)
	return rootCmd.ExecuteContext(ctx)
}
