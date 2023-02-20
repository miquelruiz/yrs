package yrs

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mmcdole/gofeed"
	ext "github.com/mmcdole/gofeed/extensions"
)

func TestNew(t *testing.T) {
	dsn := fmt.Sprintf("file:%s/yrs.db", os.TempDir())
	_, err := New("sqlite3", dsn)
	if err != nil {
		t.Error(err)
	}

	_, err = os.Stat(strings.TrimPrefix(dsn, "file:"))
	if err != nil {
		t.Error(err)
	}
}

func TestSubscribe(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO channels .+").
		ExpectExec().
		WithArgs("id", "url", "name", "rss").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectPrepare("INSERT INTO videos .+").
		ExpectExec().
		WithArgs("videoId", "link", "title", sqlmock.AnyArg(), "channelId", 0).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	feed := &gofeed.Feed{
		Items: []*gofeed.Item{
			{
				Published: "2006-01-02T15:04:05Z",
				Title:     "title",
				Link:      "link",
				Extensions: ext.Extensions{"yt": map[string][]ext.Extension{
					"videoId":   {{Value: "videoId"}},
					"channelId": {{Value: "channelId"}},
				}},
			},
		},
	}

	y := &Yrs{db}
	err = y.subscribeChannel(Channel{
		ID:           "id",
		URL:          "url",
		Name:         "name",
		RSS:          "rss",
		Autodownload: false,
	}, feed)
	if err != nil {
		t.Error(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
