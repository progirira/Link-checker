package clients

import (
	"fmt"
	"go-progira/lib/e"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

const (
	getUpdatesMethod  = "getUpdates"
	sendMessageMethod = "sendMessage"
)

type TelegramClient struct {
	host     string
	basePath string
	client   http.Client
}

func NewTelegramClient(host, token string) TelegramClient {
	return TelegramClient{
		host:     host,
		basePath: newBasePath(token),
		client:   http.Client{},
	}
}

func newBasePath(token string) string {
	return "bot" + token
}

func (c *TelegramClient) Updates(offset int, limit int) ([]byte, error) {
	q := url.Values{}
	q.Add("offset", strconv.Itoa(offset))
	q.Add("limit", strconv.Itoa(limit))

	data, err := c.doRequest(getUpdatesMethod, q)
	fmt.Println("Get response")
	fmt.Println(string(data))

	if err != nil {
		return nil, err // вставить свой метод
	}

	fmt.Println(string(data))

	return data, nil
}

func (c *TelegramClient) SendMessage(chatID int, text string) error {
	q := url.Values{}
	q.Add("chat_id", strconv.Itoa(chatID))
	q.Add("text", text)

	body, err := c.doRequest(sendMessageMethod, q)
	if err != nil {
		return e.Wrap("can't send message", err)
	}
	fmt.Println("Sending message to ", chatID)

	if body != nil {
		fmt.Printf("Response: %s\n", string(body))
	}

	return nil
}

func (c *TelegramClient) doRequest(method string, query url.Values) ([]byte, error) {
	const errMsg = "can't do request"

	u := url.URL{
		Scheme: "https",
		Host:   c.host,
		Path:   path.Join(c.basePath, method),
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), http.NoBody)
	if err != nil {
		return nil, e.Wrap(errMsg, err)
	}
	req.URL.RawQuery = query.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, e.Wrap(errMsg, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err // вставить кастомную ошибку
	}

	return body, nil
}
