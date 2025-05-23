package processing

import (
	"encoding/json"
	"go-progira/internal/application/bot/clients"
	"go-progira/internal/domain/types/bottypes"
	"go-progira/pkg/config"
	"go-progira/pkg/e"
	"log/slog"
	"net/http"
	"time"
)

type Server struct {
	tgClient *clients.TelegramClient
}

func NewServer(tgClient *clients.TelegramClient) *Server {
	return &Server{
		tgClient: tgClient,
	}
}

func (s *Server) handleUpdates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		slog.Error(
			e.ErrMethodNotAllowed.Error(),
			slog.String("method", r.Method),
			slog.String("allowed method", http.MethodPost),
		)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

		return
	}

	var linkUpdate bottypes.LinkUpdate

	if err := json.NewDecoder(r.Body).Decode(&linkUpdate); err != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", err.Error()),
		)

		sendErrorResponse(w, "Invalid JSON", "400", "LinkUpdate", "Failed to decode JSON", nil)

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	response := map[string]string{"status": "Update received"}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error(
			e.ErrEncodeToJSON.Error(),
			slog.String("error", err.Error()),
		)

		return
	}

	_ = s.tgClient.SendMessage(int(linkUpdate.TgChatIDs[0]), linkUpdate.Description+linkUpdate.URL)
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
		slog.Error(
			e.ErrEncodeToJSON.Error(),
			slog.String("error", err.Error()),
		)

		return
	}
}

func (s *Server) Start(config *config.Config) {
	http.HandleFunc("/updates", s.handleUpdates)

	srv := &http.Server{
		Addr:         config.BotHost,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("Starting bot server on",
		slog.String("address", config.BotHost))

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			slog.Error(
				e.ErrServerFailed.Error(),
				slog.String("error", err.Error()),
			)
		}
	}()
}
