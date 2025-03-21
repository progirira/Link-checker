package clients

import (
	"go-progira/lib/e"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

const (
	getUpdatesMethod  = "getUpdates"
	sendMessageMethod = "sendMessage"
)

type HTTPTelegramClient interface {
	Updates(offset, limit int) ([]byte, error)
	SendMessage(chatID int, text string) error
}

type TelegramClient struct {
	Scheme   string
	Host     string
	BasePath string
	Client   http.Client
}

func NewTelegramClient(scheme, host, token string) TelegramClient {
	return TelegramClient{
		Scheme:   scheme,
		Host:     host,
		BasePath: newBasePath(token),
		Client:   http.Client{},
	}
}

func newBasePath(token string) string {
	return "bot" + token
}

func (c *TelegramClient) Updates(offset, limit int) ([]byte, error) {
	q := url.Values{}
	q.Add("offset", strconv.Itoa(offset))
	q.Add("limit", strconv.Itoa(limit))

	body, errDoReq := DoRequest(c.Client, http.MethodGet, c.Scheme, c.Host, path.Join(c.BasePath, getUpdatesMethod), q, nil)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("method", getUpdatesMethod),
			slog.String("error", errDoReq.Error()),
		)

		return nil, errDoReq
	}

	data, errRead := io.ReadAll(body)
	if errRead != nil {
		slog.Error(
			e.ErrReadBody.Error(),
			slog.String("error", errRead.Error()))

		return nil, errRead
	}

	return data, nil
}

func (c *TelegramClient) SendMessage(chatID int, text string) error {
	q := url.Values{}
	q.Add("chat_id", strconv.Itoa(chatID))
	q.Add("text", text)

	_, errDoReq := DoRequest(c.Client, http.MethodGet, c.Scheme, c.Host, path.Join(c.BasePath, sendMessageMethod), q, nil)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("method", getUpdatesMethod),
			slog.String("error", errDoReq.Error()),
		)

		return errDoReq
	}

	return nil
}
