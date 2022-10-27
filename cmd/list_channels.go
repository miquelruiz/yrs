package cmd

import (
	"database/sql"
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
	ctx := cmd.Context()
	db := ctx.Value(DbKey).(*sql.DB)
	rows, err := db.Query("SELECT * FROM channels")
	if err != nil {
		return fmt.Errorf("couldn't retrieve the channels: %w", err)
	}
	defer rows.Close()

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', 0)
	fmt.Fprintln(w, "ID\tName\tURL\tAutodownload")
	for rows.Next() {
		c := schema.Channel{}
		err = rows.Scan(&c.ID, &c.URL, &c.Name, &c.RSS, &c.Autodownload)
		if err != nil {
			return fmt.Errorf("scan failed: %w", err)
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%t\t\n", c.ID, c.Name, c.URL, c.Autodownload)
	}

	w.Flush()
	return nil
}
