package clients

import (
	"go-progira/lib/e"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

const (
	getUpdatesMethod  = "getUpdates"
	sendMessageMethod = "sendMessage"
)

type MyClient interface {
	DoRequest(method string, query url.Values) ([]byte, error)
	SendMessage(chatID int, text string) error
}

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

func (c *TelegramClient) Updates(offset, limit int) ([]byte, error) {
	q := url.Values{}
	q.Add("offset", strconv.Itoa(offset))
	q.Add("limit", strconv.Itoa(limit))

	body, err := DoRequest(c.client, getUpdatesMethod, c.host, c.basePath, q, nil)
	data, _ := io.ReadAll(body)

	//fmt.Println("Get response")
	//fmt.Println(string(data))

	if err != nil {
		return nil, err
	}

	return data, nil
}

func (c *TelegramClient) SendMessage(chatID int, text string) error {
	q := url.Values{}
	q.Add("chat_id", strconv.Itoa(chatID))
	q.Add("text", text)

	_, err := DoRequest(c.client, sendMessageMethod, c.host, c.basePath, q, nil)
	if err != nil {
		return e.Wrap("can't send message", err)
	}
	//fmt.Println("Sending message to ", chatID)

	//if body != nil {
	//	fmt.Printf("Response: %s\n", string(body))
	//}

	return nil
}
