package main

import (
	"database/sql"
	"errors"
	"fmt"
	"go-progira/internal/application/scrapper"
	sqldatabase "go-progira/internal/repository/sql_database"
	"go-progira/pkg"
	"go-progira/pkg/config"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	pkg.SetNewStdoutLogger()

	envData, errLoadEnv := config.Set(".env")
	if errLoadEnv != nil {
		return
	}

	connString, err := envData.GetByKeyFromEnv("DATABASE_URL")

	storage, err := sqldatabase.NewSQLStorage(connString)
	if errors.Is(err, sqldatabase.ErrPoolCreate) {
		fmt.Println(err)

		return
	}

	conn, err := sql.Open("postgres", connString)
	defer conn.Close()
	migrator := sqldatabase.MustGetNewMigrator()
	err = migrator.ApplyMigrations(conn)
	if err != nil {
		log.Printf("Migrations error %e", err)
		panic(err)
	} else {
		log.Printf("Migrations applied!!")
	}

	//st := &dictionary_storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	botClient := scrapper.NewBotClient("http", "bot:8090", "/updates")
	scr := scrapper.NewServer(storage, botClient)

	batchStr, errLoad := envData.GetByKeyFromEnv("BATCH")
	if errLoad != nil {
		return
	}
	batch, _ := strconv.Atoi(batchStr)
	scr.Start(batch)
	log.Println("Batch", batch)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("Shutting down...")
}
