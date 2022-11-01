package main

import "github.com/miquelruiz/youtube-rss-subscriber-go/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
