package clients

import (
	"encoding/json"
	"fmt"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/lib/e"
	"log/slog"
	"net/http"
	"net/url"
)

type ScrapperClient struct {
	client http.Client
	scheme string
	host   string
}

func NewScrapperClient(scheme, host string) ScrapperClient {
	return ScrapperClient{
		client: http.Client{},
		scheme: scheme,
		host:   host,
	}
}

func (c *ScrapperClient) RegisterChat(id int64) {
	c.doWithChat(http.MethodPost, id, "error registering chat: %s")
}

func (c *ScrapperClient) DeleteChat(id int64) {
	c.doWithChat(http.MethodDelete, id, "error deleting chat: %s")
}

func (c *ScrapperClient) doWithChat(method string, id int64, debugMes string) {
	u := fmt.Sprintf("/tg-chat/%d", id)

	_, err := DoRequest(c.client, method, "http", c.host, u, url.Values{}, nil)
	if err != nil {
		slog.Debug(
			e.ErrDoRequest.Error(),
			slog.String("message", debugMes),
		)
	}
}

func (c *ScrapperClient) GetLinks(chatID int64) (*scrappertypes.ListLinksResponse, error) {
	u := "/links"

	q := url.Values{}
	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

	responseBody, errDoReq := DoRequest(c.client, http.MethodGet, c.scheme, c.host, u, q, nil)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return nil, e.ErrDoRequest
	}

	var listResp scrappertypes.ListLinksResponse
	if errDecode := json.NewDecoder(responseBody).Decode(&listResp); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return nil, e.ErrDecodeJSONBody
	}

	return &listResp, nil
}

func (c *ScrapperClient) AddLink(chatID int64, request scrappertypes.AddLinkRequest) (*scrappertypes.LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()))

		return nil, e.ErrMarshalJSON
	}

	return c.doWithLink(http.MethodPost, chatID, body)
}

func (c *ScrapperClient) RemoveLink(chatID int64, request scrappertypes.RemoveLinkRequest) (*scrappertypes.LinkResponse, error) {
	body, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()))

		return nil, e.ErrMarshalJSON
	}

	return c.doWithLink(http.MethodDelete, chatID, body)
}

func (c *ScrapperClient) doWithLink(method string, chatID int64, body []byte) (*scrappertypes.LinkResponse, error) {
	u := "/links"

	q := url.Values{}

	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))
	q.Add("Content-Type", "application/json")

	responseBody, errDoReq := DoRequest(c.client, method, c.scheme, c.host, u, q, body)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return nil, e.ErrDoRequest
	}

	var linkResponse scrappertypes.LinkResponse

	if ErrDecode := json.NewDecoder(responseBody).Decode(&linkResponse); ErrDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", ErrDecode.Error()),
		)

		return nil, e.ErrDecodeJSONBody
	}

	return &linkResponse, nil
}
