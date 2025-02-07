package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/miquelruiz/yrs/internal/config"
	"github.com/miquelruiz/yrs/pkg/yrs"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"golang.org/x/tools/blog/atom"
)

const (
	ENTRIES_IN_FEED             = 40
	DEFAULT_UPDATE_INTERVAL_SEC = 3600
)

var (
	rootUrl    string
	configPath string
	address    string
	port       int
)

type WebYrs yrs.Yrs

func init() {
	flag.StringVar(&rootUrl, "root-url", "", "Root of the URL where the app will be served")
	flag.StringVar(&configPath, "config", "/etc/yrs/config.yml", "Path to the config file")
	flag.StringVar(&address, "address", "127.0.0.1", "Address to bind to")
	flag.IntVar(&port, "port", 8080, "Port to bind to")
	flag.Parse()
	cleanRootUrl()
}

func createRender() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	r.AddFromFiles("index", "templates/base.tmpl")
	r.AddFromFiles("listChannels", "templates/base.tmpl", "templates/channels.tmpl")
	r.AddFromFiles("videos", "templates/base.tmpl", "templates/videos.tmpl")
	return r
}

func (w *WebYrs) listChannels(c *gin.Context) {
	var err error
	y := yrs.Yrs(*w)
	errStr := c.Query("error")
	if errStr != "" {
		err = errors.New(errStr)
	}
	channels, errGet := y.GetChannels()
	if err != nil || errGet != nil {
		err = errors.Join(err, errGet)
	}
	c.HTML(http.StatusOK, "listChannels", gin.H{
		"rootUrl":  rootUrl,
		"channels": channels,
		"error":    err,
	})
}

func (w *WebYrs) getVideos(vGetter func() ([]yrs.Video, error), n int) ([]yrs.Video, error) {
	videos, err := vGetter()
	if n != 0 && len(videos) > n {
		videos = videos[len(videos)-n:]
	}
	return videos, err
}

func (w *WebYrs) listVideos(c *gin.Context) {
	var parseErr error
	lastInt := 0
	last := c.DefaultQuery("last", "20")
	if last != "all" {
		lastInt, parseErr = strconv.Atoi(last)
	}

	var newVideos []yrs.Video
	var updateErr error
	showNew := false
	if c.Request.Method == "POST" {
		y := yrs.Yrs(*w)
		newVideos, updateErr = y.Update()
		showNew = true
	}

	var videos []yrs.Video
	var getVErr error
	y := yrs.Yrs(*w)
	channel := c.DefaultQuery("channel", "")
	if channel != "" {
		videos, getVErr = w.getVideos(
			func() ([]yrs.Video, error) { return y.GetVideosByChannel(channel) },
			lastInt,
		)
	} else {
		videos, getVErr = w.getVideos(y.GetVideos, lastInt)
	}

	c.HTML(http.StatusOK, "videos", gin.H{
		"rootUrl":   rootUrl,
		"videos":    videos,
		"newVideos": len(newVideos),
		"showNew":   showNew,
		"error":     errors.Join(updateErr, getVErr, parseErr),
	})
}

func (w *WebYrs) generateFeed(c *gin.Context) {
	feed := atom.Feed{
		Title: "YouTube RSS Subscriber",
		ID:    "yrs",
	}

	y := yrs.Yrs(*w)
	videos, err := w.getVideos(y.GetVideos, ENTRIES_IN_FEED)
	if err != nil {
		c.XML(500, feed)
		return
	}

	feed.Entry = lo.Map(videos, func(v yrs.Video, _ int) *atom.Entry {
		return &atom.Entry{
			Title:     v.Title,
			ID:        v.ID,
			Link:      []atom.Link{{Href: v.URL}},
			Published: atom.TimeStr(v.Published.String()),
			Author:    &atom.Person{Name: v.Channel.Name},
		}
	})

	c.XML(200, feed)
}

func (w *WebYrs) search(c *gin.Context) {
	y := yrs.Yrs(*w)
	results, err := y.Search(c.Query("term"))

	var videos []yrs.Video
	if err == nil {
		ids := lo.Map(results, func(r yrs.SearchResult, _ int) string {
			return r.ID
		})
		videos, err = y.GetVideosByID(ids)
	}

	c.HTML(http.StatusOK, "videos", gin.H{
		"show_update": false,
		"rootUrl":     rootUrl,
		"videos":      videos,
		"error":       err,
	})
}

func (w *WebYrs) subscribeYouTube(c *gin.Context) {
	var errArg string
	y := yrs.Yrs(*w)
	err := y.SubscribeYouTubeID(c.PostForm("channelID"))
	if err != nil {
		errArg = fmt.Sprintf("?error=%s", url.QueryEscape(err.Error()))
	}
	c.Redirect(303, buildUrl("/list-channels")+errArg)
}

func (w *WebYrs) subscribe(c *gin.Context) {
	var errArg string
	y := yrs.Yrs(*w)
	err := y.Subscribe(c.PostForm("rss"))
	if err != nil {
		errArg = fmt.Sprintf("?error=%s", url.QueryEscape(err.Error()))
	}
	c.Redirect(303, buildUrl("/list-channels")+errArg)
}

func index(c *gin.Context) {
	c.HTML(http.StatusOK, "index", gin.H{"rootUrl": rootUrl})
}

func buildUrl(url string) string {
	return fmt.Sprintf("%s%s", rootUrl, url)
}

func cleanRootUrl() {
	if rootUrl == "" {
		return
	}
	rootUrl = fmt.Sprintf(
		"/%s",
		strings.TrimSuffix(strings.TrimPrefix(rootUrl, "/"), "/"),
	)
}

func runWebServer(wy *WebYrs) *http.Server {
	r := gin.Default()
	r.HTMLRender = createRender()

	r.Static(buildUrl("/js/"), "js")
	r.Static(buildUrl("/css/"), "css")

	r.GET(buildUrl("/list-channels"), wy.listChannels)
	r.GET(buildUrl("/list-videos"), wy.listVideos)
	r.POST(buildUrl("/list-videos"), wy.listVideos)

	r.POST(buildUrl("/subscribeYouTube"), wy.subscribeYouTube)
	r.POST(buildUrl("/subscribe"), wy.subscribe)

	r.GET(buildUrl("/feed"), wy.generateFeed)
	r.GET(buildUrl("/search"), wy.search)

	r.GET(buildUrl("/"), index)

	addr := fmt.Sprintf("%s:%d", address, port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	return srv
}

func runUpdater(wy *WebYrs, quit chan os.Signal) *time.Ticker {
	y := yrs.Yrs(*wy)
	ticker := time.NewTicker(DEFAULT_UPDATE_INTERVAL_SEC * time.Second)
	go func() {
		for {
			select {
			case <-quit:
				return
			case <-ticker.C:
				if _, err := y.Update(); err != nil {
					log.Println(err)
				}
			}
		}
	}()
	return ticker
}

func main() {
	config, err := config.Load(configPath)
	if err != nil {
		panic(err)
	}

	y, err := yrs.New(config.DatabaseDriver, config.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	wy := WebYrs(*y)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	srv := runWebServer(&wy)
	ticker := runUpdater(&wy, quit)
	defer ticker.Stop()

	// Block until a signal is received.
	<-quit
	log.Println("Shutting down...")
	close(quit)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}

	log.Println("Done")
}
