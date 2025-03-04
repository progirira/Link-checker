package bot

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/application/bot/clients"
	bottypes "go-progira/internal/domain/types/bot_types"
	"log"
	"net/http"
	"time"
)

type Server struct {
	TgClient clients.TelegramClient
}

func handleUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var linkUpdate bottypes.LinkUpdate

	if err := json.NewDecoder(r.Body).Decode(&linkUpdate); err != nil {
		log.Println("Error decoding JSON:", err)
		sendErrorResponse(w, "Invalid JSON", "400", "LinkUpdate", "Failed to decode JSON", nil)

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{"status": "Update received"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Println("Error encoding response:", err)
	}

	fmt.Println("Update received")
}

func sendErrorResponse(w http.ResponseWriter, desc, code, exceptionName, exceptionMsg string, stacktrace []string) {
	apiError := bottypes.APIErrorResponse{
		Description:      desc,
		Code:             code,
		ExceptionName:    exceptionName,
		ExceptionMessage: exceptionMsg,
		Stacktrace:       stacktrace,
	}

	w.WriteHeader(http.StatusBadRequest)
	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(apiError); err != nil {
		log.Println("Error encoding error response:", err)
	}
}

func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/updates", handleUpdates)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("Starting server on :8080...")

	if err := srv.ListenAndServe(); err != nil {
		fmt.Println("Server failed:", err)
	}
}
