package main

import (
	"fmt"
	"log"

	"github.com/miquelruiz/yrs/lib"
	"github.com/miquelruiz/yrs/yrs"

	"net/http"
)

type WebYrs yrs.Yrs

func (w *WebYrs) listChannels(rw http.ResponseWriter, req *http.Request) {
	y := yrs.Yrs(*w)
	channels, err := y.GetChannels()
	if err != nil {
		fmt.Fprintf(rw, "Error retrieving channels: %v", err)
		return
	}

	for i, c := range channels {
		fmt.Fprintf(
			rw,
			"%d\t%s\t%s\t%s\t%t\t\n",
			i,
			c.ID,
			c.Name,
			c.URL,
			c.Autodownload,
		)
	}
}

func main() {
	config, err := lib.LoadConfig("")
	if err != nil {
		panic(err)
	}
	fmt.Println(config)
	y, err := yrs.New(config.DatabaseDriver, config.DatabaseUrl)
	if err != nil {
		panic(err)
	}

	wy := WebYrs(*y)
	http.HandleFunc("/list-channels", wy.listChannels)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
