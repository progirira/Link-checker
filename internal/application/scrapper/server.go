package scrapper

import (
	"encoding/json"
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
	chats     map[int64]*scrappertypes.Chat
	chatMutex sync.Mutex
	botClient BotClient
}

func NewServer(client BotClient) *Server {
	return &Server{
		chats:     make(map[int64]*scrappertypes.Chat),
		chatMutex: sync.Mutex{},
		botClient: client,
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

func (s *Server) updateLinkContent(chatID int64, url, lastContent string) {
	for i, l := range s.chats[chatID].Links {
		if url == l.URL {
			if lastContent != "" && lastContent != l.LastVersion {
				s.chats[chatID].Links[i].LastVersion = lastContent
				s.chats[chatID].Links[i].LastChecked = time.Now()

				update := bottypes.LinkUpdate{
					ID:          l.ID,
					URL:         l.URL,
					Description: botmessages.MsgUpdatesHappened,
					TgChatIDs:   []int64{chatID},
				}

				err := s.botClient.sendUpdate(update)
				if err != nil {
					return
				}
			}
		}
	}
}

func (s *Server) monitorLinks() {
	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	for chatID, chat := range s.chats {
		for _, link := range chat.Links {
			switch IsStackOverflowURL(link.URL) {
			case true:
				currentVersion, err := GetStackOverflowUpdates(link.URL)
				if err != nil {
					return
				}

				if currentVersion != "" {
					s.updateLinkContent(chatID, link.URL, currentVersion)
				}
			case false:
				if IsGitHubURL(link.URL) {
					currentVersion, err := CheckGitHubUpdates(link.URL)
					if err != nil {
						return
					}

					if currentVersion != "" {
						s.updateLinkContent(chatID, link.URL, currentVersion)
					}
				}

			default:
				slog.Error(
					e.ErrWrongURLFormat.Error(),
					slog.String("function", "monitorLinks"),
					slog.String("url", link.URL),
				)
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

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	if _, exists := s.chats[id]; exists {
		http.Error(w, "Chat already exists.", http.StatusBadRequest)
		return
	}

	s.chats[id] = &scrappertypes.Chat{ID: id, Links: []scrappertypes.LinkResponse{}}
	response := map[string]interface{}{"message": "Chat registered successfully", "id": id}

	w.WriteHeader(http.StatusOK)

	errEncoding := json.NewEncoder(w).Encode(response)
	if errEncoding != nil {
		return
	}
}

func (s *Server) DeleteChat(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/tg-chat/"):]

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	if _, exists := s.chats[id]; !exists {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	delete(s.chats, id)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetLinks(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil || chatID <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	chat, exists := s.chats[chatID]
	if !exists {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	response1 := scrappertypes.ListLinksResponse{Links: chat.Links, Size: len(chat.Links)}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response1)
	if err != nil {
		return
	}
}

func (s *Server) AddLink(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil || chatID <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var request scrappertypes.AddLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request.", http.StatusBadRequest)
		return
	}

	content := ""

	if IsGitHubURL(request.Link) {
		content, err = CheckGitHubUpdates(request.Link)
		if err != nil {
			return
		}
	} else if IsStackOverflowURL(request.Link) {
		content, err = GetStackOverflowUpdates(request.Link)
		if err != nil {
			return
		}
	}

	link := &scrappertypes.LinkResponse{
		ID:          int64(len(s.chats[chatID].Links) + 1),
		URL:         request.Link,
		Tags:        request.Tags,
		Filters:     request.Filters,
		LastVersion: content,
		LastChecked: time.Now(),
	}

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	chat, exists := s.chats[chatID]
	if !exists {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	if !s.isURLInAdded(chatID, link.URL) {
		chat.Links = append(chat.Links, *link)
	} else {
		update := bottypes.LinkUpdate{
			ID:          link.ID,
			URL:         link.URL,
			Description: botmessages.MsgAlreadyExists,
			TgChatIDs:   []int64{chatID},
		}

		err := s.botClient.sendUpdate(update)
		if err != nil {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
}

func (s *Server) isURLInAdded(id int64, u string) bool {
	if len(s.chats[id].Links) == 0 {
		return false
	}

	for _, l := range s.chats[id].Links {
		if l.URL == u {
			return true
		}
	}

	return false
}

func (s *Server) RemoveLink(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.URL.Query().Get("Tg-Chat-Id")
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil || chatID <= 0 {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	var request scrappertypes.RemoveLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request.", http.StatusBadRequest)
		return
	}

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	chat, exists := s.chats[chatID]
	if !exists {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	for i, link := range chat.Links {
		if link.URL == request.Link {
			chat.Links = append(chat.Links[:i], chat.Links[i+1:]...)
			return
		}
	}

	http.Error(w, "Link not found.", http.StatusNotFound)
}
