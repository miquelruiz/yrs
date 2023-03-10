package yrs

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/mmcdole/gofeed"
	ext "github.com/mmcdole/gofeed/extensions"
)

func mustCreateYrs(t *testing.T) *Yrs {
	tempdir, err := os.MkdirTemp(os.TempDir(), "*")
	if err != nil {
		t.Fatal(err)
	}
	dsn := fmt.Sprintf("file:%s/yrs.db", tempdir)
	t.Cleanup(func() {
		os.RemoveAll(tempdir)
	})

	y, err := New("sqlite3", dsn)
	if err != nil {
		t.Fatal(err)
	}

	_, err = os.Stat(strings.TrimPrefix(dsn, "file:"))
	if err != nil {
		t.Fatal(err)
	}

	return y
}

func setupFixtures(y *Yrs) error {
	feed := &gofeed.Feed{
		Items: []*gofeed.Item{
			{
				Published: "2006-01-02T15:04:05Z",
				Title:     "title",
				Link:      "link",
				Extensions: ext.Extensions{"yt": map[string][]ext.Extension{
					"videoId":   {{Value: "videoId"}},
					"channelId": {{Value: "id"}},
				}},
			},
		},
	}

	return y.subscribeChannel(Channel{
		ID:           "id",
		URL:          "url",
		Name:         "name",
		RSS:          "rss",
		Autodownload: false,
	}, feed)
}

func TestChannel(t *testing.T) {
	y := mustCreateYrs(t)
	err := setupFixtures(y)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		got func(Channel) string
		exp string
	}{
		{got: func(c Channel) string { return c.ID }, exp: "id"},
		{got: func(c Channel) string { return c.URL }, exp: "url"},
		{got: func(c Channel) string { return c.Name }, exp: "name"},
		{got: func(c Channel) string { return c.RSS }, exp: "rss"},
	}

	channels, err := y.GetChannels()
	if err != nil {
		t.Error(err)
	}

	if len(channels) != 1 {
		t.Fatalf("Unexpected number of channels. Got %d, Expected %d", len(channels), 1)
	}

	for _, test := range testCases {
		got := test.got(channels[0])
		if got != test.exp {
			t.Errorf("Unexpected channel value. Got: %s Expected: %s", got, test.exp)
		}
	}
}

func TestVideos(t *testing.T) {
	y := mustCreateYrs(t)
	err := setupFixtures(y)
	if err != nil {
		t.Fatal(err)
	}

	testCases := []struct {
		got func(Video) string
		exp string
	}{
		{got: func(v Video) string { return v.ID }, exp: "videoId"},
		{got: func(v Video) string { return v.Title }, exp: "title"},
		{got: func(v Video) string { return v.ChannelId }, exp: "id"},
	}

	videos, err := y.GetVideos()
	if err != nil {
		t.Error(err)
	}

	if len(videos) != 1 {
		t.Fatalf("Unexpected number of videos. Got %d, Expected %d", len(videos), 1)
	}

	for _, test := range testCases {
		got := test.got(videos[0])
		if got != test.exp {
			t.Errorf("Unexpected video value. Got: %s Expected: %s", got, test.exp)
		}
	}
}

func TestSearch(t *testing.T) {
	y := mustCreateYrs(t)
	err := setupFixtures(y)
	if err != nil {
		t.Fatal(err)
	}

	r, err := y.Search("title")
	if err != nil {
		t.Fatal(err)
	}

	if len(r) != 1 {
		t.Fatalf("Unexpected number of search results. Got %d, Expected %d", len(r), 1)
	}

	if r[0].ID != "videoId" || r[0].Title != "title" || r[0].Channel != "name" {
		t.Fatalf("Unexpected search result. Got %s, Expected {videoId title name}", r)
	}
}
