package scrapper

import (
	"bytes"
	"encoding/json"
	bottypes "go-progira/internal/domain/types/bot_types"
	"go-progira/pkg/e"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

type HTTPBotClient interface {
	SendUpdate(update bottypes.LinkUpdate) (err error)
}

type BotClient struct {
	scheme   string
	host     string
	basePath string
}

func NewBotClient(scheme, host, basePath string) *BotClient {
	return &BotClient{
		scheme:   scheme,
		host:     host,
		basePath: basePath,
	}
}

func (c *BotClient) SendUpdate(update bottypes.LinkUpdate) (err error) {
	u := url.URL{
		Scheme: c.scheme,
		Host:   c.host,
		Path:   c.basePath,
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()),
		)

		return e.ErrMarshalJSON
	}

	resp, errDoReq := http.Post(u.String(), "application/json", bytes.NewBuffer(jsonData))

	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
			slog.String("url", u.String()),
		)

		return e.ErrDoRequest
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		var apiError bottypes.APIErrorResponse

		if errDecode := json.Unmarshal(body, &apiError); errDecode != nil {
			slog.Error(
				e.ErrDecodeJSONBody.Error(),
				slog.String("error", errDecode.Error()),
				slog.String("response", string(body)),
			)
		}

		slog.Error(
			e.ErrAPI.Error(),
			slog.String("error", apiError.Description),
		)

		return e.ErrAPI
	}

	errClose := resp.Body.Close()
	if errClose != nil {
		slog.Error(
			e.ErrCloseBody.Error(),
			slog.String("error", errClose.Error()),
		)

		return e.ErrCloseBody
	}

	return nil
}
