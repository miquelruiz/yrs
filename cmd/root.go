package cmd

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
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
	db, err := sql.Open("sqlite3", "yrs.db")
	if err != nil {
		return fmt.Errorf("couldn't open the database: %w", err)
	}

	ctx := context.WithValue(context.Background(), DbKey, db)
	return rootCmd.ExecuteContext(ctx)
}
