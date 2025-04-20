package scrapper

import (
	"context"
	"encoding/json"
	"errors"
	"go-progira/internal/application/scrapper/api"
	botmessages "go-progira/internal/domain/bot_messages"
	"go-progira/internal/domain/types/api_types"
	bottypes "go-progira/internal/domain/types/bot_types"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/internal/formatter"
	"go-progira/internal/repository/dictionary_storage"
	"go-progira/pkg/config"
	"go-progira/pkg/e"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-co-op/gocron"
)

type Server struct {
	Storage   repository.Storage
	BotClient HTTPBotClient
}

func NewServer(storage repository.Storage, client HTTPBotClient) *Server {
	return &Server{
		Storage:   storage,
		BotClient: client,
	}
}

func (s *Server) Start(batch int) {
	http.HandleFunc("/tg-chat/{id}", s.ChatHandler)
	http.HandleFunc("/links", s.LinksHandler)
	s.startScheduler(batch)

	slog.Debug("Starting server on :8090...")

	srv := &http.Server{
		Addr:         ":8090",
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

func (s *Server) monitorLinks(batch int) {
	ctx := context.Background()

	links, lastID := s.Storage.GetBatchOfLinks(ctx, batch, int64(0))

	for len(links) != 0 {
		for _, link := range links {
			prevTime := s.Storage.GetPreviousUpdate(ctx, link.ID)

			var msg string

			if api.IsStackOverflowURL(link.URL) {
				envData, errLoadEnv := config.Set(".env")
				if errLoadEnv != nil {
					return
				}

				apiKey, _ := envData.GetByKeyFromEnv("STACKOVERFLOW_API_KEY")
				updater := api.StackoverflowUpdater{Key: apiKey}

				updates, _ := s.saveStackoverflowUpdates(ctx, updater, link.ID, link.URL, prevTime)

				if len(updates) == 0 {
					continue
				}

				msg = formatter.FormatMessageForStackOverflow(updates)
				slog.Info("Got %d updates", len(updates))
			} else {
				updater := api.GithubUpdater{}
				updates, _ := s.saveGithubUpdates(ctx, updater, link.ID, link.URL, prevTime)

				if len(updates) == 0 {
					continue
				}

				msg = formatter.FormatMessageForGithub(updates)
				slog.Info("Got %d updates", len(updates))
			}

			IDs := s.Storage.GetTgChatIDsForLink(ctx, link.URL)

			if len(IDs) == 0 {
				continue
			}

			updForBot := bottypes.LinkUpdate{
				URL:         link.URL,
				Description: msg,
				TgChatIDs:   IDs,
			}

			err := s.BotClient.SendUpdate(updForBot)
			if err != nil {
				return
			}
		}
		links, lastID = s.Storage.GetBatchOfLinks(ctx, batch, lastID)
	}
}

func (s *Server) saveStackoverflowUpdates(ctx context.Context, updater api.StackoverflowUpdater, linkID int64, url string, prevTime time.Time) ([]api_types.StackOverFlowUpdate, error) {
	updates := updater.GetUpdates(url, prevTime)

	for _, update := range updates {
		t := time.Unix(update.CreatedAt, 0)

		err := s.Storage.SaveLastUpdate(ctx, linkID, t)
		if err != nil {
			return []api_types.StackOverFlowUpdate{}, err
		}
	}

	return updates, nil
}

func (s *Server) saveGithubUpdates(ctx context.Context, updater api.GithubUpdater, linkID int64, url string, prevTime time.Time) ([]api_types.GithubUpdate, error) {
	//log.Println("In saveGithubUpdates function")
	updates := updater.GetUpdates(url, prevTime)

	for _, update := range updates {
		t, err := time.Parse(time.RFC3339, update.CreatedAt)
		if err != nil {
			return []api_types.GithubUpdate{}, err
		}

		errSave := s.Storage.SaveLastUpdate(ctx, linkID, t)
		if errSave != nil {
			return []api_types.GithubUpdate{}, errSave
		}
	}

	return updates, nil
}

func (s *Server) startScheduler(batch int) {
	slog.Info("Scheduler started")

	sc := gocron.NewScheduler(time.UTC)

	_, err := sc.Every(10).Minutes().Do(func() {
		go s.monitorLinks(batch)
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
			slog.String("error", err.Error()))
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
			slog.String("error", err.Error()))

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
