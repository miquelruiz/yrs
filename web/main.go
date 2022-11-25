package main

import (
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"

	"github.com/miquelruiz/yrs/lib"
	"github.com/miquelruiz/yrs/yrs"

	"net/http"
)

type WebYrs yrs.Yrs

var (
	//go:embed css js
	static embed.FS

	//go:embed templates
	templates embed.FS

	address string
	port    int
)

func init() {
	flag.StringVar(&address, "address", "127.0.0.1", "Address to bind to")
	flag.IntVar(&port, "port", 8080, "Port to bind to")
	flag.Parse()
}

func (w *WebYrs) listChannels(rw http.ResponseWriter, req *http.Request) {
	y := yrs.Yrs(*w)
	channels, err := y.GetChannels()
	if err != nil {
		fmt.Fprintf(rw, "Error retrieving channels: %v", err)
		return
	}

	t, _ := template.ParseFS(templates, "templates/base.tmpl", "templates/channels.tmpl")
	if err = t.Execute(rw, channels); err != nil {
		fmt.Println(err)
		return
	}
}

func renderVideos(rw http.ResponseWriter, req *http.Request, videos []yrs.Video) {
	t, _ := template.ParseFS(templates, "templates/base.tmpl", "templates/videos.tmpl")
	if err := t.Execute(rw, videos); err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	config, err := lib.LoadConfig("")
	if err != nil {
		panic(err)
	}

	y, err := yrs.New(config.DatabaseDriver, config.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	wy := WebYrs(*y)

	mux := http.NewServeMux()

	mux.Handle("/css/", http.FileServer(http.FS(static)))
	mux.Handle("/js/", http.FileServer(http.FS(static)))

	mux.HandleFunc("/list-channels", wy.listChannels)
	mux.HandleFunc("/list-videos", func(rw http.ResponseWriter, req *http.Request) {
		videos, err := y.GetVideos()
		if err != nil {
			fmt.Fprintf(rw, "Error retrieving videos: %v", err)
			return
		}
		renderVideos(rw, req, videos)
	})

	mux.HandleFunc("/update", func(rw http.ResponseWriter, req *http.Request) {
		videos, err := y.Update()
		if err != nil {
			fmt.Fprintf(rw, "Error updating videos: %v", err)
			return
		}
		renderVideos(rw, req, videos)
	})

	mux.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		t, err := template.ParseFS(templates, "templates/base.tmpl")
		if err != nil {
			fmt.Printf("Error parsing template: %v", err)
			return
		}
		if err = t.Execute(rw, nil); err != nil {
			fmt.Printf("Error running template: %v", err)
			return
		}
	})

	addr := fmt.Sprintf("%s:%d", address, port)
	fmt.Printf("Serving at %s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
