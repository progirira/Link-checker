package scrapper

import (
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	botmessages "go-progira/internal/domain/bot_messages"
	bottypes "go-progira/internal/domain/types/bot_types"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const MaxInt32 = 2147483647

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

	fmt.Println("Starting server on :8090...")

	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Println("Server failed:", err)
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
			var currentVersion string

			switch IsStackOverflowURL(link.URL) {
			case true:
				currentVersion, _ = GetStackOverflowUpdates(link.URL)

				s.updateLinkContent(chatID, link.URL, currentVersion)
			case false:
				if IsGitHubURL(link.URL) {
					// currentVersion, _ = CheckGitHubUpdates(link.URL)
				}

			default:
				fmt.Println("не github и не stackoverflow")
			}
		}
	}
}

func (s *Server) startScheduler() {
	sc := gocron.NewScheduler(time.UTC)
	task := s.monitorLinks

	_, err := sc.Every(6).Seconds().Do(task)
	if err != nil {
		return
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

	size := len(chat.Links)
	if size > MaxInt32 {
		http.Error(w, "Too many links", http.StatusInternalServerError)
		return
	}

	response1 := scrappertypes.ListLinksResponse{Links: chat.Links, Size: int32(len(chat.Links))}

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

	content, err := GetStackOverflowUpdates(request.Link)

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
