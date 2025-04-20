package main

import (
	"go-progira/internal/application/scrapper"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/internal/repository/storage"
	"go-progira/pkg"
)

func main() {
	pkg.SetNewStdoutLogger()

	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	botClient := scrapper.NewBotClient("http", "localhost:8080", "/updates")
	scr := scrapper.NewServer(dict, botClient)

	scr.Start()
}
