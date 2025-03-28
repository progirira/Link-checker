package main

import (
	"errors"
	"fmt"
	"go-progira/internal/application/scrapper"
	"go-progira/internal/application/scrapper/storage"
	"go-progira/internal/domain"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/lib/e"
)

func main() {
	fileForLogs := "logs/scrapper_logs.txt"

	loggerErr := domain.SetNewLogger(fileForLogs)
	if errors.Is(loggerErr, e.ErrOpenFile) {
		fmt.Println(loggerErr.Error())

		return
	}

	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	botClient := scrapper.NewBotClient("http", "localhost:8080", "/updates")
	scr := scrapper.NewServer(dict, botClient)

	scr.Start()
}
