package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/miquelruiz/youtube-rss-subscriber-go/schema"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listChannelsCmd)
}

var listChannelsCmd = &cobra.Command{
	Use:   "list-channels",
	Short: "List all the subscribed channels",
	RunE:  listChannels,
}

func listChannels(cmd *cobra.Command, args []string) error {
	db := cmd.Context().Value(DbKey).(*schema.Schema)

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "ID\tName\tURL\tAutodownload")
	err := db.ForEachChannel(func(c *schema.Channel) {
		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t\n", c.ID, c.Name, c.URL, c.Autodownload)
	})

	if err != nil {
		return fmt.Errorf("failed to list channels: %w", err)
	}

	return nil
}
