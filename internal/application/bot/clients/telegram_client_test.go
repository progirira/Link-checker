package clients_test

import (
	"errors"
	"go-progira/internal/application/bot/clients"
	"go-progira/lib/e"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTelegramClient_SendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &clients.TelegramClient{
		Client:   http.Client{},
		Scheme:   "http",
		Host:     server.Listener.Addr().String(),
		BasePath: "/botTOKEN",
	}

	err := client.SendMessage(12345, "Hello!")
	if err != nil {
		t.Errorf("Wrong error. Expected: %v, Got: %v", nil, err)
	}
}

func TestTelegramClient_SendMessage_Error(t *testing.T) {
	client := &clients.TelegramClient{
		Client:   http.Client{},
		Scheme:   "http",
		Host:     "invalid_host",
		BasePath: "/botTOKEN",
	}

	err := client.SendMessage(12345, "Hello!")
	if !errors.Is(err, e.ErrDoRequest) {
		t.Errorf("Wrong error. Expected: %v, Got: %v", e.ErrDoRequest, err)
	}
}
