package clients

import (
	"encoding/json"
	"fmt"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"net/http"
	"net/url"
)

type ScrapperClient struct {
	client http.Client
	host   string
}

func NewScrapperClient(host string) ScrapperClient {
	return ScrapperClient{
		client: http.Client{},
		host:   host,
	}
}

//func (c *ScrapperClient) createURLString(p string) string {
//	return path.Join(c.BaseURL, p)
//}

func (c *ScrapperClient) RegisterChat(id int64) error {
	return c.doWithChat(http.MethodPost, id, "error registering chat: %s")
}

func (c *ScrapperClient) DeleteChat(id int64) error {
	return c.doWithChat(http.MethodDelete, id, "error deleting chat: %s")
}

func (c *ScrapperClient) doWithChat(method string, id int64, debugMes string) error {
	u := fmt.Sprintf("/tg-chat/%d", id)

	_, err := DoRequest(c.client, method, c.host, u, url.Values{}, nil)
	if err != nil {
		fmt.Println(debugMes)
		return err
	}

	return nil
}

func (c *ScrapperClient) GetLinks(chatID int64) (*scrappertypes.ListLinksResponse, error) {
	u := fmt.Sprintf("/links")
	fmt.Println("GetLinks ScrapClient")

	q := url.Values{}
	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

	responseBody, err := DoRequest(c.client, http.MethodGet, c.host, u, q, nil)

	if err != nil {
		return nil, err
	}

	var linksResponse scrappertypes.ListLinksResponse
	if err := json.NewDecoder(responseBody).Decode(&linksResponse); err != nil {
		return nil, err
	}
	fmt.Println(&linksResponse)
	return &linksResponse, nil
}

func (c *ScrapperClient) AddLink(chatID int64, request scrappertypes.AddLinkRequest) (*scrappertypes.LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	return c.doWithLink(http.MethodPost, chatID, body)
}

func (c *ScrapperClient) RemoveLink(chatID int64, request scrappertypes.RemoveLinkRequest) (*scrappertypes.LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	return c.doWithLink(http.MethodDelete, chatID, body)
}

func (c *ScrapperClient) doWithLink(method string, chatID int64, body []byte) (*scrappertypes.LinkResponse, error) {
	u := "/links"

	//req, err := http.NewRequest(method, url, bytes.NewBuffer(body))

	q := url.Values{}
	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))
	q.Add("Content-Type", "application/json")

	responseBody, err := DoRequest(c.client, method, c.host, u, q, body)

	if err != nil {
		return nil, err
	}

	var linkResponse scrappertypes.LinkResponse
	if err := json.NewDecoder(responseBody).Decode(&linkResponse); err != nil {
		return nil, err
	}

	return &linkResponse, nil
}
