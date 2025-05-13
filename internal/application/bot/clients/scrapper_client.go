package clients

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/domain/types/scrappertypes"
	"go-progira/pkg/e"
	"log/slog"
	"net/http"
	"net/url"
)

type HTTPScrapperClient interface {
	RegisterChat(id int64)
	DeleteChat(id int64)
	AddLink(chatID int64, request scrappertypes.AddLinkRequest) error
	GetLinks(chatID int64) (*scrappertypes.ListLinksResponse, error)
	RemoveLink(chatID int64, request scrappertypes.RemoveLinkRequest) error
	GetLinksByTag(chatID int64, request scrappertypes.GetLinksByTagsRequest) (*scrappertypes.ListLinksResponse, error)
	DeleteTag(chatID int64, request scrappertypes.DeleteTagRequest) error
}

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

	response, err := DoRequest(c.client, method, "http", c.host, u, url.Values{}, nil, false)
	if err != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("message", debugMes),
		)
	}

	if response == nil {
		return
	}

	errClose := response.Body.Close()
	if errClose != nil {
		slog.Error("Error closing response body" + errClose.Error())
	}
}

func (c *ScrapperClient) GetLinks(chatID int64) (*scrappertypes.ListLinksResponse, error) {
	u := "/links"

	q := url.Values{}
	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

	response, errDoReq := DoRequest(c.client, http.MethodGet, c.scheme, c.host, u, q, nil, false)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return nil, e.ErrDoRequest
	}

	var listResp scrappertypes.ListLinksResponse
	if errDecode := json.NewDecoder(response.Body).Decode(&listResp); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return nil, e.ErrDecodeJSONBody
	}

	errClose := response.Body.Close()
	if errClose != nil {
		slog.Error("Error closing response body" + errClose.Error())

		return nil, errClose
	}

	return &listResp, nil
}

func (c *ScrapperClient) GetLinksByTag(chatID int64, request scrappertypes.GetLinksByTagsRequest) (
	*scrappertypes.ListLinksResponse, error) {
	u := "/tags"

	q := url.Values{}
	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

	body, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()))

		return nil, e.ErrMarshalJSON
	}

	response, errDoReq := DoRequest(c.client, http.MethodGet, c.scheme, c.host, u, q, body, false)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return nil, e.ErrDoRequest
	}

	var listResp scrappertypes.ListLinksResponse
	if errDecode := json.NewDecoder(response.Body).Decode(&listResp); errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)

		return nil, e.ErrDecodeJSONBody
	}

	errClose := response.Body.Close()
	if errClose != nil {
		slog.Error("Error closing response body" + errClose.Error())

		return &listResp, errClose
	}

	return &listResp, nil
}

func (c *ScrapperClient) DeleteTag(chatID int64, request scrappertypes.DeleteTagRequest) error {
	u := "/tags"

	q := url.Values{}
	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

	body, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()))

		return e.ErrMarshalJSON
	}

	response, errDoReq := DoRequest(c.client, http.MethodDelete, c.scheme, c.host, u, q, body, false)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return e.ErrDoRequest
	}

	errClose := response.Body.Close()
	if errClose != nil {
		slog.Error("Error closing response body" + errClose.Error())

		return errClose
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			return e.ErrTagNotFound
		} else if response.StatusCode == http.StatusInternalServerError {
			return e.ErrDeleteTag
		}

		slog.Error("Unknown error while deleting tag: ",
			slog.Int("status code", response.StatusCode))
	}

	return nil
}

func (c *ScrapperClient) AddLink(chatID int64, request scrappertypes.AddLinkRequest) error {
	body, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()))

		return e.ErrMarshalJSON
	}

	response, err := c.doWithLink(http.MethodPost, chatID, body)
	if err != nil {
		return e.ErrAddLink
	}

	if response == nil {
		return nil
	}

	errClose := response.Body.Close()
	if errClose != nil {
		slog.Error("Error closing response body" + errClose.Error())

		return errClose
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusConflict {
			return e.ErrLinkAlreadyExists
		}

		return e.ErrAddLink
	}

	return nil
}

func (c *ScrapperClient) RemoveLink(chatID int64, request scrappertypes.RemoveLinkRequest) error {
	body, err := json.Marshal(request)
	if err != nil {
		slog.Error(
			e.ErrMarshalJSON.Error(),
			slog.String("error", err.Error()))

		return e.ErrMarshalJSON
	}

	response, err := c.doWithLink(http.MethodDelete, chatID, body)

	if err != nil {
		return e.ErrDeleteLink
	}

	if response == nil {
		return nil
	}

	errClose := response.Body.Close()
	if errClose != nil {
		slog.Error("Error closing response body" + errClose.Error())
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusNotFound {
			return e.ErrLinkNotFound
		} else if response.StatusCode == http.StatusInternalServerError {
			return e.ErrDeleteLink
		}
	}

	return err
}

func (c *ScrapperClient) doWithLink(method string, chatID int64, body []byte) (*http.Response, error) {
	u := "/links"

	q := url.Values{}

	q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))
	q.Add("Content-Type", "application/json")

	response, errDoReq := DoRequest(c.client, method, c.scheme, c.host, u, q, body, false)
	if errDoReq != nil {
		slog.Error(
			e.ErrDoRequest.Error(),
			slog.String("error", errDoReq.Error()),
		)

		return response, e.ErrDoRequest
	}

	if response == nil {
		return nil, nil
	}

	if response.StatusCode != http.StatusOK {
		slog.Error("Error while doing operation with link:",
			slog.Int("status code", response.StatusCode))
	}

	return response, nil
}
