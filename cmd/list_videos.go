package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/miquelruiz/youtube-rss-subscriber-go/schema"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listVideosCmd)
}

var listVideosCmd = &cobra.Command{
	Use:   "list-videos",
	Short: "List all the videos in the database",
	RunE:  listVideos,
}

func listVideos(cmd *cobra.Command, args []string) error {
	db := cmd.Context().Value(DbKey).(*schema.Schema)

	w := tabwriter.NewWriter(os.Stdout, 5, 2, 3, ' ', 0)
	defer w.Flush()

	fmt.Fprintln(w, "ID\tTitle\tURL\tPublished\tChannelId")
	err := db.ForEachVideo(func(v *schema.Video) {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t\n", v.ID, v.Title, v.URL, v.Published, v.ChannelId)
	})

	return err
}
