package cmd

import (
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "yrs",
		Short: "YouTube RSS Subscriber",
		Long:  "A tool to subscribe to YouTube channels without a YouTube account",
	}
)

func Execute() error {
	return rootCmd.Execute()
}
