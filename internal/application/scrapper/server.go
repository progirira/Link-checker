package scrapper

import (
	"encoding/json"
	"errors"
	"go-progira/internal/application/scrapper/api"
	"go-progira/internal/application/scrapper/storage"
	botmessages "go-progira/internal/domain/bot_messages"
	bottypes "go-progira/internal/domain/types/bot_types"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/lib/e"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

type Server struct {
	Storage   storage.Storage
	ChatMutex sync.Mutex
	BotClient HTTPBotClient
}

func NewServer(storage storage.Storage, client HTTPBotClient) *Server {
	return &Server{
		Storage:   storage,
		ChatMutex: sync.Mutex{},
		BotClient: client,
	}
}

func (s *Server) Start() {
	http.HandleFunc("/tg-chat/{id}", s.ChatHandler)
	http.HandleFunc("/links", s.LinksHandler)
	s.startScheduler()

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

func (s *Server) monitorLinks() {
	s.ChatMutex.Lock()
	defer s.ChatMutex.Unlock()

	updates := s.Storage.Update()
	if len(updates) > 0 {
		for _, upd := range updates {
			err := s.BotClient.SendUpdate(upd)
			if err != nil {
				return
			}
		}
	}
}

func (s *Server) startScheduler() {
	sc := gocron.NewScheduler(time.UTC)
	task := s.monitorLinks

	_, err := sc.Every(6).Seconds().Do(task)
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
	idStr := r.URL.Path[len("/tg-chat/"):]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	s.ChatMutex.Lock()
	defer s.ChatMutex.Unlock()

	errCreate := s.Storage.CreateChat(id)
	if errCreate != nil {
		http.Error(w, "Chat already exists.", http.StatusBadRequest)
	} else {
		response := map[string]interface{}{"message": "Chat registered successfully", "id": id}

		w.WriteHeader(http.StatusOK)

		errEncoding := json.NewEncoder(w).Encode(response)
		if errEncoding != nil {
			return
		}
	}
}

func (s *Server) DeleteChat(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/tg-chat/"):]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	s.ChatMutex.Lock()
	defer s.ChatMutex.Unlock()

	errDelete := s.Storage.DeleteChat(id)
	if errDelete != nil {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetLinks(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")

	id, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	s.ChatMutex.Lock()
	defer s.ChatMutex.Unlock()

	links, errGet := s.Storage.GetLinks(id)

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
	slog.Debug("In add Link in scrapper server")

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

	content := ""

	if api.IsGitHubURL(request.Link) {
		content, err = api.CheckGitHubUpdates(request.Link)
		if err != nil {
			return
		}
	} else if api.IsStackOverflowURL(request.Link) {
		content, err = api.GetStackOverflowUpdates(request.Link)
		if err != nil {
			return
		}
	}

	errAppend := s.Storage.AddLink(id, request.Link, request.Tags, request.Filters, content)
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
	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")
	id, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil || id <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var request scrappertypes.RemoveLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request.", http.StatusBadRequest)
		return
	}

	s.ChatMutex.Lock()
	defer s.ChatMutex.Unlock()

	errRemove := s.Storage.RemoveLink(id, request.Link)
	if errRemove == nil {
		return
	}

	if errors.Is(errRemove, e.ErrChatNotFound) {
		http.Error(w, "Chat not found.", http.StatusNotFound)
	} else {
		http.Error(w, "Link not found.", http.StatusNotFound)
	}
}
