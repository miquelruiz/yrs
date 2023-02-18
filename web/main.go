package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"strconv"
	"strings"

	"github.com/miquelruiz/yrs/lib"
	"github.com/miquelruiz/yrs/yrs"

	"net/http"
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
}

func (w *WebYrs) listChannels(rw http.ResponseWriter, req *http.Request) {
	fmt.Println("Serving /list-channels")
	y := yrs.Yrs(*w)
	channels, err := y.GetChannels()
	if err != nil {
		fmt.Fprintf(rw, "Error retrieving channels: %v", err)
		return
	}

	t, err := template.ParseFiles("templates/base.tmpl", "templates/channels.tmpl")
	if err != nil {
		fmt.Println(err)
		return
	}

	tplVars := map[string]interface{}{
		"rootUrl":  rootUrl,
		"channels": channels,
	}
	if err = t.Execute(rw, tplVars); err != nil {
		fmt.Println(err)
		return
	}
}

func renderVideos(rw http.ResponseWriter, req *http.Request, videos []yrs.Video) {
	t, err := template.ParseFiles("templates/base.tmpl", "templates/videos.tmpl")
	if err != nil {
		fmt.Println(err)
		return
	}

	tplVars := map[string]interface{}{
		"rootUrl": rootUrl,
		"videos":  videos,
	}
	if err := t.Execute(rw, tplVars); err != nil {
		fmt.Println(err)
		return
	}
}

func buildUrl(url string) string {
	cleanRoot := strings.TrimSuffix(rootUrl, "/")
	return fmt.Sprintf("%s%s", cleanRoot, url)
}

func main() {
	config, err := lib.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}

	y, err := yrs.New(config.DatabaseDriver, config.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	wy := WebYrs(*y)
	mux := http.NewServeMux()

	mux.Handle(buildUrl("/css/"), http.StripPrefix(buildUrl("/css/"), http.FileServer(http.Dir("css"))))
	mux.Handle(buildUrl("/js/"), http.StripPrefix(buildUrl("/js/"), http.FileServer(http.Dir("js"))))

	mux.HandleFunc(buildUrl("/list-channels"), wy.listChannels)
	mux.HandleFunc(buildUrl("/list-videos"), func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Serving /list-videos")
		videos, err := y.GetVideos()
		if err != nil {
			fmt.Fprintf(rw, "Error retrieving videos: %v", err)
			return
		}
		last := req.URL.Query().Get("last")
		if last != "" {
			lastInt, err := strconv.Atoi(last)
			if err != nil {
				fmt.Printf("Ilegal 'last' param: %s", err)
				lastInt = len(videos)
			}
			if len(videos) > lastInt {
				videos = videos[len(videos)-lastInt:]
			}
		}
		renderVideos(rw, req, videos)
	})

	mux.HandleFunc(buildUrl("/update"), func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Serving /update")
		videos, err := y.Update()
		if err != nil {
			fmt.Fprintf(rw, "Error updating videos: %v", err)
			return
		}
		renderVideos(rw, req, videos)
	})

	mux.HandleFunc(buildUrl("/"), func(rw http.ResponseWriter, req *http.Request) {
		fmt.Println("Serving /")
		t, err := template.ParseFiles("templates/base.tmpl")
		if err != nil {
			fmt.Printf("Error parsing template: %v", err)
			return
		}

		tplVars := map[string]interface{}{
			"rootUrl": rootUrl,
		}
		if err = t.Execute(rw, tplVars); err != nil {
			fmt.Printf("Error running template: %v", err)
			return
		}
	})

	addr := fmt.Sprintf("%s:%d", address, port)
	fmt.Printf("Serving at %s/%s\n", addr, rootUrl)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
