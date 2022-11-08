package main

import "github.com/miquelruiz/yrs/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
