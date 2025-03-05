package scrapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	bottypes "go-progira/internal/domain/types/bot_types"
	"io"
	"net/http"
	"net/url"
)

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

func (c *BotClient) sendUpdate(update bottypes.LinkUpdate) error {
	u := url.URL{
		Scheme: c.scheme,
		Host:   c.host,
		Path:   c.basePath,
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %s", err)
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(jsonData))
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	if err != nil {
		return fmt.Errorf("failed to send request: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		var apiError bottypes.APIErrorResponse

		if err := json.Unmarshal(body, &apiError); err != nil {
			return fmt.Errorf("non-200 status code: %s, response: %s", resp.Status, string(body))
		}

		return fmt.Errorf("API returned error: %s", apiError.Description)
	}

	return nil
}
