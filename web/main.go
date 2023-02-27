package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/miquelruiz/yrs/internal/config"
	"github.com/miquelruiz/yrs/pkg/yrs"

	"github.com/gin-contrib/multitemplate"
	"github.com/gin-gonic/gin"
)

type WebYrs yrs.Yrs

var (
	rootUrl    string
	configPath string
	address    string
	port       int
)

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

func (w *WebYrs) listVideos(c *gin.Context) {
	y := yrs.Yrs(*w)
	videos, err := y.GetVideos()
	last := c.DefaultQuery("last", "20")
	if last != "all" {
		lastInt, err := strconv.Atoi(last)
		if err != nil {
			lastInt = len(videos)
		}
		if len(videos) > lastInt {
			videos = videos[len(videos)-lastInt:]
		}
	}
	c.HTML(http.StatusOK, "videos", gin.H{
		"show_update": false,
		"rootUrl":     rootUrl,
		"videos":      videos,
		"error":       err,
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
	r := gin.Default()
	r.HTMLRender = createRender()

	r.Static(buildUrl("/js/"), "js")
	r.Static(buildUrl("/css/"), "css")

	r.GET(buildUrl("/list-channels"), wy.listChannels)
	r.GET(buildUrl("/list-videos"), wy.listVideos)

	r.GET(buildUrl("/update"), wy.update)
	r.POST(buildUrl("/update"), wy.update)

	r.GET(buildUrl("/"), index)

	addr := fmt.Sprintf("%s:%d", address, port)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
