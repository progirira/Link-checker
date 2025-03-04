package scrapper

import (
	"bytes"
	"encoding/json"
	"fmt"
	bottypes "go-progira/internal/domain/types/bot_types"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
)

type BotClient struct{}

func (c *BotClient) sendUpdate(update bottypes.LinkUpdate) error {
	u := url.URL{
		Scheme: "http",
		Host:   "://localhost:",
		Path:   path.Join("8080", http.MethodPost),
	}

	jsonData, err := json.Marshal(update)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %s", err)
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %s", err)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		var apiError bottypes.APIErrorResponse

		if err := json.Unmarshal(body, &apiError); err != nil {
			return fmt.Errorf("non-200 status code: %s, response: %s", resp.Status, string(body))
		}

		return fmt.Errorf("API returned error: %s", apiError.Description)
	}

	log.Println("Update sent successfully!")

	return nil
}
