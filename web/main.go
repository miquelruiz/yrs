package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/miquelruiz/yrs/internal/config"
	"github.com/miquelruiz/yrs/pkg/yrs"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
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
	y := yrs.Yrs(*w)
	channels, err := y.GetChannels()
	c.HTML(http.StatusOK, "listChannels", gin.H{
		"rootUrl":  rootUrl,
		"channels": channels,
		"error":    err,
	})
}

func (w *WebYrs) getVideos(n int) ([]yrs.Video, error) {
	y := yrs.Yrs(*w)
	videos, err := y.GetVideos()
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
	videos, err := w.getVideos(lastInt)
	c.HTML(http.StatusOK, "videos", gin.H{
		"show_update": false,
		"rootUrl":     rootUrl,
		"videos":      videos,
		"error":       errors.Join(err, parseErr),
	})
}

func (w *WebYrs) update(c *gin.Context) {
	var videos []yrs.Video
	var err error
	show_no_new := false
	if c.Request.Method == "POST" {
		y := yrs.Yrs(*w)
		videos, err = y.Update()
		show_no_new = true
	}
	c.HTML(http.StatusOK, "videos", gin.H{
		"show_update": true,
		"show_no_new": show_no_new,
		"rootUrl":     rootUrl,
		"videos":      videos,
		"error":       err,
	})
}

func (w *WebYrs) generateFeed(c *gin.Context) {
	feed := atom.Feed{
		Title: "YouTube RSS Subscriber",
		ID:    "yrs",
	}

	videos, err := w.getVideos(ENTRIES_IN_FEED)
	if err != nil {
		c.XML(500, feed)
		return
	}

	for _, v := range videos {
		entry := atom.Entry{
			Title:     v.Title,
			ID:        v.ID,
			Link:      []atom.Link{{Href: v.URL}},
			Published: atom.TimeStr(v.Published.String()),
			Author:    &atom.Person{Name: v.Channel.Name},
		}
		feed.Entry = append(feed.Entry, &entry)
	}

	c.XML(200, feed)
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

	r.GET(buildUrl("/update"), wy.update)
	r.POST(buildUrl("/update"), wy.update)

	r.GET(buildUrl("/feed"), wy.generateFeed)

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
