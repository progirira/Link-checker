package scrapper

import (
	"encoding/json"
	"fmt"
	"github.com/go-co-op/gocron"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Server struct {
	chats     map[int64]*scrappertypes.Chat // Хранение чатов в памяти
	chatMutex sync.Mutex                    // Мьютекс для защиты доступа к мапе чатов
	botClient BotClient
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/tg-chat/", s.ChatHandler)
	mux.HandleFunc("/links", s.LinksHandler)

	s.startScheduler()

	fmt.Println("Starting server on :8090...")
	if err := http.ListenAndServe(":8090", nil); err != nil {
		fmt.Println("Server failed:", err)
	}
}

func (s *Server) monitorLinks() {
	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()

	for _, chat := range s.chats {
		for _, link := range chat.Links {
			var currentVersion string

			switch IsStackOverflowURL(link.URL) {
			case true:
				currentVersion, _ = CheckStackOverflowUpdates(link.URL)
			case false:
				if IsGitHubURL(link.URL) {
					currentVersion, _ = CheckGitHubUpdates(link.URL)
				}
			default:
				fmt.Println("не гитхаб и не стековерфлоу")
			}

			if currentVersion != link.LastVersion {
				fmt.Printf("Changes detected for link %s\n", link.URL)
				link.LastVersion = currentVersion
				link.LastChecked = time.Now()
			}
		}
	}
}

func (s *Server) startScheduler() {
	sc := gocron.NewScheduler(time.UTC)

	task := s.monitorLinks

	_, err := sc.Every(5).Minutes().Do(task)
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
	fmt.Println("register")
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

	w.WriteHeader(http.StatusOK)
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

	fmt.Println("In Get Links Server")
	chatIDStr := r.Header.Get("Tg-Chat-Id")

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

	response := scrappertypes.ListLinksResponse{Links: chat.Links, Size: int32(len(chat.Links))}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		return
	}
}

func (s *Server) AddLink(w http.ResponseWriter, r *http.Request) {
	fmt.Println("In Add Links Server")

	chatIDStr := r.Header.Get("Tg-Chat-Id")
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

	link := scrappertypes.LinkResponse{
		ID:      int64(len(s.chats[chatID].Links) + 1), // Простой ID генератор
		URL:     request.Link,
		Tags:    request.Tags,
		Filters: request.Filters,
	}

	s.chatMutex.Lock()
	defer s.chatMutex.Unlock()
	chat, exists := s.chats[chatID]
	if !exists {
		http.Error(w, "Chat not found.", http.StatusNotFound)
		return
	}

	if !s.isURLInAdded(link.ID, link.URL) {
		chat.Links = append(chat.Links, link)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(link)
}

func (s *Server) isURLInAdded(id int64, u string) bool {
	for _, l := range s.chats[id].Links {
		if l.URL == u {
			return true
		}
	}
	return false
}

func (s *Server) RemoveLink(w http.ResponseWriter, r *http.Request) {
	chatIDStr := r.Header.Get("Tg-Chat-Id")
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
			json.NewEncoder(w).Encode(link)

			return
		}
	}

	http.Error(w, "Link not found.", http.StatusNotFound)
}
