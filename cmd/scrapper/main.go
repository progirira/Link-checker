package main

import "go-progira/internal/application/scrapper"

func main() {
	scr := scrapper.NewServer()
	scr.Start()
}
