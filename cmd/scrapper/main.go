package main

import "go-progira/internal/application/scrapper"

func main() {
	botClient := scrapper.NewBotClient("http", "localhost:8080", "/updates")
	scr := scrapper.NewServer(*botClient)

	scr.Start()
}
