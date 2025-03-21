package main

import (
	"errors"
	"fmt"
	"go-progira/internal/application/scrapper"
	"go-progira/internal/domain"
	"go-progira/lib/e"
)

func main() {
	fileForLogs := "logs/scrapper_logs"

	loggerErr := domain.SetNewLogger(fileForLogs)
	if errors.Is(loggerErr, e.ErrOpenFile) {
		fmt.Println(loggerErr.Error())

		return
	}

	botClient := scrapper.NewBotClient("http", "localhost:8080", "/updates")
	scr := scrapper.NewServer(botClient)

	scr.Start()
}
