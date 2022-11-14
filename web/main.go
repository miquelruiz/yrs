package main

import (
	"embed"
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
)

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

func (w *WebYrs) listVideos(rw http.ResponseWriter, req *http.Request) {
	y := yrs.Yrs(*w)
	videos, err := y.GetVideos()
	if err != nil {
		fmt.Fprintf(rw, "Error retrieving videos: %v", err)
		return
	}

	t, _ := template.ParseFS(templates, "templates/base.tmpl", "templates/videos.tmpl")
	if err = t.Execute(rw, videos); err != nil {
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
	mux.HandleFunc("/list-videos", wy.listVideos)

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

	fmt.Println("Serving")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
