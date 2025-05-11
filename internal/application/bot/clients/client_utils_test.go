package clients_test

import (
	"go-progira/internal/application/bot/clients"
	"go-progira/pkg/e"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDoRequest_Success(t *testing.T) {
	mockResponse := `{"status": "ok"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte(mockResponse))
		if err != nil {
			slog.Error(
				e.ErrWrite.Error(),
				slog.String("error", err.Error()),
			)
		}
	}))

	defer server.Close()

	client := http.Client{}

	parsedURL, _ := url.Parse(server.URL)

	resp, err := clients.DoRequest(client, http.MethodGet, parsedURL.Scheme, parsedURL.Host, "/",
		nil, nil, false)
	assert.NoError(t, err)

	body, err := io.ReadAll(resp.Body)

	assert.NoError(t, err)
	assert.Equal(t, mockResponse, string(body))

	err = resp.Body.Close()
	assert.NoError(t, err)
}

func TestDoRequest_BadURL(t *testing.T) {
	client := http.Client{}

	resp, err := clients.DoRequest(client, http.MethodGet, "http", ":", "/",
		nil, nil, false)
	assert.Error(t, err)

	assert.Nil(t, resp)

	if resp != nil && resp.Body != nil {
		err = resp.Body.Close()
		assert.NoError(t, err)
	}
}

func TestDoRequest_ClientError(t *testing.T) {
	brokenClient := http.Client{
		Transport: &http.Transport{
			Proxy: func(*http.Request) (*url.URL, error) {
				return nil, assert.AnError
			},
		},
	}

	resp, err := clients.DoRequest(brokenClient, http.MethodGet, "http", "localhost", "/",
		nil, nil, false)
	assert.Error(t, err)

	assert.Nil(t, resp)

	if resp != nil && resp.Body != nil {
		err = resp.Body.Close()
		assert.NoError(t, err)
	}
}
