package scrapper

import (
	"context"
	"encoding/json"
	"errors"
	"go-progira/internal/application/scrapper/api"
	"go-progira/internal/domain/botmessages"
	"go-progira/internal/domain/types/bottypes"
	"go-progira/internal/domain/types/scrappertypes"
	repository "go-progira/internal/repository/dictionary_storage"
	"go-progira/pkg/config"
	"go-progira/pkg/e"
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

type Server struct {
	Storage   repository.LinkService
	BotClient HTTPBotClient
}

func NewServer(storage repository.LinkService, client HTTPBotClient) *Server {
	return &Server{
		Storage:   storage,
		BotClient: client,
	}
}

func (s *Server) Start(config *config.Config) {
	http.HandleFunc("/tg-chat/{id}", s.ChatHandler)
	http.HandleFunc("/links", s.LinksHandler)
	s.startScheduler(config)

	slog.Info("Starting scrapper server on",
		slog.String("address", config.ScrapperHost))

	srv := &http.Server{
		Addr:         config.ScrapperHost,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		slog.Error(
			e.ErrServerFailed.Error(),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Server) processLink(ctx context.Context, link *scrappertypes.LinkResponse) {
	prevTime := s.Storage.GetPreviousUpdate(ctx, link.ID)

	var lastUpdateTime time.Time

	var msg string

	if updater, ok := api.GetUpdater(link.URL); ok {
		msg, lastUpdateTime = updater.GetUpdates(link.URL, prevTime)
		if msg == "" {
			return
		}
	} else {
		slog.Error(
			e.ErrWrongURLFormat.Error(),
			slog.String("url", link.URL),
		)

		return
	}

	errSave := s.saveLastUpdate(ctx, link.ID, lastUpdateTime)
	if errSave != nil {
		slog.Error("Error saving update")

		return
	}

	IDs := s.Storage.GetTgChatIDsForLink(ctx, link.URL)

	if len(IDs) == 0 {
		return
	}

	updForBot := bottypes.LinkUpdate{
		URL:         link.URL,
		Description: msg,
		TgChatIDs:   IDs,
	}

	errSend := s.BotClient.SendUpdate(updForBot)
	if errSend != nil {
		return
	}
}

func splitIntoChunks(links []scrappertypes.LinkResponse, numChunks int) [][]scrappertypes.LinkResponse {
	var chunks [][]scrappertypes.LinkResponse

	chunkSize := int(math.Ceil(float64(len(links)) / float64(numChunks)))

	for i := 0; i < len(links); i += chunkSize {
		end := i + chunkSize
		if end > len(links) {
			end = len(links)
		}

		chunks = append(chunks, links[i:end])
	}

	return chunks
}

func (s *Server) processChunk(ctx context.Context, chunk []scrappertypes.LinkResponse, wg *sync.WaitGroup) {
	defer wg.Done()

	for _, link := range chunk {
		s.processLink(ctx, &link)
	}
}

func (s *Server) monitorLinks(config *config.Config) {
	ctx := context.Background()

	if config.Workers <= 0 {
		slog.Error("Invalid number of workers, it must be greater than zero",
			slog.Int("given number of workers", config.Workers),
			slog.String("action", "not processing batches, returning from monitor links"))

		return
	}

	links, lastID := s.Storage.GetBatchOfLinks(ctx, config.Batch, int64(0))

	api.InitUpdaters(config.StackoverflowAPIKey)

	for len(links) != 0 {
		chunks := splitIntoChunks(links, config.Workers)

		var wg sync.WaitGroup

		wg.Add(config.Workers)

		for _, chunk := range chunks {
			go s.processChunk(ctx, chunk, &wg)
		}

		wg.Wait()

		links, lastID = s.Storage.GetBatchOfLinks(ctx, config.Batch, lastID)
	}
}

func (s *Server) saveLastUpdate(ctx context.Context, linkID int64, lastUpdateTime time.Time) error {
	err := s.Storage.SaveLastUpdate(ctx, linkID, lastUpdateTime)
	if err != nil {
		slog.Error("Failed to save last update time",
			slog.Int("link id", int(linkID)))

		return err
	}

	return nil
}

func (s *Server) startScheduler(config *config.Config) {
	slog.Info("Scheduler started")

	sc := gocron.NewScheduler(time.UTC)

	_, err := sc.Every(10).Minutes().Do(func() {
		go s.monitorLinks(config)
	})
	if err != nil {
		slog.Error(
			e.ErrScheduler.Error(),
			slog.String("error", err.Error()),
		)
	}

	sc.StartAsync()
}

func (s *Server) LinksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.GetLinks(w, r)
	case http.MethodPost:
		s.AddLink(w, r)
	case http.MethodDelete:
		s.RemoveLink(w, r)
	default:
		slog.Error(
			e.ErrMethodNotAllowed.Error(),
			slog.String("method", r.Method),
			slog.String("allowed methods", strings.Join([]string{http.MethodGet, http.MethodPost, http.MethodDelete}, " ")),
		)

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) ChatHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.RegisterChat(w, r)
	case http.MethodDelete:
		s.DeleteChat(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) RegisterChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.URL.Path[len("/tg-chat/"):]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	errCreate := s.Storage.CreateChat(ctx, id)
	if errCreate != nil {
		slog.Error("Error creating chate",
			slog.String("error", errCreate.Error()))
		http.Error(w, "Chat already exists.", http.StatusBadRequest)

		return
	}

	slog.Info("Registered chat")

	response := map[string]interface{}{"message": "Chat registered successfully", "id": id}

	w.WriteHeader(http.StatusOK)

	errEncoding := json.NewEncoder(w).Encode(response)
	if errEncoding != nil {
		return
	}
}

func (s *Server) DeleteChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := r.URL.Path[len("/tg-chat/"):]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	errDelete := s.Storage.DeleteChat(ctx, id)
	if errDelete != nil {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetLinks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")

	id, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	links, errGet := s.Storage.GetLinks(ctx, id)
	if errGet != nil {
		slog.Error("Error getting link",
			slog.String("error", errGet.Error()))
		http.Error(w, "Chat not found.", http.StatusNotFound)
	} else {
		response1 := scrappertypes.ListLinksResponse{Links: links, Size: len(links)}

		w.Header().Set("Content-Type", "application/json")

		err = json.NewEncoder(w).Encode(response1)
		if err != nil {
			return
		}
	}
}

func (s *Server) AddLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	slog.Info("In add Link in scrapper server")

	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")
	id, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var request scrappertypes.AddLinkRequest
	if errDecode := json.NewDecoder(r.Body).Decode(&request); errDecode != nil {
		http.Error(w, "Invalid request.", http.StatusBadRequest)
		return
	}

	errAppend := s.Storage.AddLink(ctx, id, request.Link, request.Tags, request.Filters)
	if errAppend == nil {
		return
	}

	if errors.Is(errAppend, e.ErrChatNotFound) {
		http.Error(w, e.ErrChatNotFound.Error(), http.StatusNotFound)
		return
	}

	if errors.Is(errAppend, e.ErrLinkAlreadyExists) {
		update := bottypes.LinkUpdate{
			ID:          id,
			URL:         request.Link,
			Description: botmessages.MsgAlreadyExists,
			TgChatIDs:   []int64{id},
		}

		err := s.BotClient.SendUpdate(update)
		if err != nil {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) RemoveLink(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")
	id, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil {
		slog.Error("Error parsing id string",
			slog.String("error", err.Error()))
		return
	}

	if id <= 0 {
		slog.Error("Error invalid id(less than zero)",
			slog.Int("id", int(id)))

		http.Error(w, "Invalid chat ID", http.StatusBadRequest)

		return
	}

	var request scrappertypes.RemoveLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request.", http.StatusBadRequest)

		return
	}

	errRemove := s.Storage.RemoveLink(ctx, id, request.Link)
	if errRemove == nil {
		return
	}

	if errors.Is(errRemove, e.ErrChatNotFound) {
		http.Error(w, "Chat not found.", http.StatusNotFound)
	} else {
		http.Error(w, "Link not found.", http.StatusNotFound)
	}
}
