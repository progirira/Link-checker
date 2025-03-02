package main

import "go-progira/internal/application/scrapper"

func main() {
	scr := scrapper.Server{}
	scr.Start()
}
