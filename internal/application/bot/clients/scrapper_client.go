package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"net/http"
	"time"
)

type ScrapperClient struct {
	client  http.Client
	Timeout time.Duration
	BaseURL string
}

func NewScrapperClient(timeout time.Duration, baseURL string) ScrapperClient {
	return ScrapperClient{
		client:  http.Client{},
		Timeout: timeout,
		BaseURL: baseURL,
	}
}

func (c *ScrapperClient) RegisterChat(id int64) error {
	path := fmt.Sprintf("%s/tg-chat/%d", c.BaseURL, id)

	response, err := http.Post(path, "application/json", nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error registering chat: %s", response.Status)
	}

	return nil
}

func (c *ScrapperClient) DeleteChat(id int64) error {
	url := fmt.Sprintf("%s/tg-chat/%d", c.BaseURL, id)

	req, err := http.NewRequest(http.MethodDelete, url, http.NoBody)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error deleting chat: %s", response.Status)
	}

	return nil
}

func (c *ScrapperClient) GetLinks(chatID int64) (*scrappertypes.ListLinksResponse, error) {
	url := fmt.Sprintf("%s/links", c.BaseURL)

	req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

	response, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error getting links: %s", response.Status)
	}

	var linksResponse scrappertypes.ListLinksResponse
	if err := json.NewDecoder(response.Body).Decode(&linksResponse); err != nil {
		return nil, err
	}

	return &linksResponse, nil
}

func (c *ScrapperClient) AddLink(chatID int64, request scrappertypes.AddLinkRequest) (*scrappertypes.LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	return c.changeLink(http.MethodPost, chatID, body)
}

func (c *ScrapperClient) RemoveLink(chatID int64, request scrappertypes.RemoveLinkRequest) (*scrappertypes.LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	return c.changeLink(http.MethodDelete, chatID, body)
}

func (c *ScrapperClient) changeLink(method string, chatID int64, body []byte) (*scrappertypes.LinkResponse, error) {
	url := fmt.Sprintf("%s/links", c.BaseURL)

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Tg-Chat-Id", fmt.Sprintf("%d", chatID))
	req.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error removing link: %s", response.Status)
	}

	var linkResponse scrappertypes.LinkResponse
	if err := json.NewDecoder(response.Body).Decode(&linkResponse); err != nil {
		return nil, err
	}

	return &linkResponse, nil
}
