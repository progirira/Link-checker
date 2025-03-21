package scrapper_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go-progira/internal/application/scrapper"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/lib/e"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterChat(t *testing.T) {
	s := &scrapper.Server{
		Chats:     make(map[int64]*scrappertypes.Chat),
		ChatMutex: sync.Mutex{},
	}

	chatID := int64(12345)

	req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(chatID, 10), http.NoBody)
	w := httptest.NewRecorder()

	s.RegisterChat(w, req)

	resp := w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error(
				e.ErrCloseBody.Error(),
				slog.String("error", err.Error()),
			)
		}
	}(resp.Body)

	expectedCode := http.StatusOK
	if resp.StatusCode != expectedCode {
		t.Errorf("Incorrect status code, got: %v; expected %v", resp.StatusCode, expectedCode)
	}

	var response map[string]interface{}
	errDecode := json.NewDecoder(resp.Body).Decode(&response)
	if errDecode != nil {
		slog.Error(
			e.ErrDecodeJSONBody.Error(),
			slog.String("error", errDecode.Error()),
		)
	}

	expectedAnswer := "Chat registered successfully"
	if response["message"] != expectedAnswer {
		t.Errorf("Incorrect message, got: %v; expected %v", response["message"], expectedAnswer)
	}

	_, exists := s.Chats[chatID]
	if !exists {
		t.Errorf("Chat not registered")
	}

	w = httptest.NewRecorder()
	s.RegisterChat(w, req) // пробуем повторно зарегистрировать чат

	resp = w.Result()
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			slog.Error(
				e.ErrCloseBody.Error(),
				slog.String("error", err.Error()),
			)
		}
	}(resp.Body)

	expectedCode = http.StatusBadRequest
	if resp.StatusCode != expectedCode {
		t.Errorf("Incorrect status code, got: %v; expected %v", resp.StatusCode, expectedCode)
	}
}

func TestIsURLInAdded(t *testing.T) {
	s := &scrapper.Server{
		Chats:     make(map[int64]*scrappertypes.Chat),
		ChatMutex: sync.Mutex{},
	}

	chatID := int64(12345)
	u := "url"
	req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(chatID, 10), http.NoBody)
	w := httptest.NewRecorder()

	s.RegisterChat(w, req)

	if s.IsURLInAdded(chatID, u) != false {
		t.Errorf("Incorrect value, got: %v; expected %v", s.IsURLInAdded(chatID, u), false)
	}

	chat := s.Chats[chatID]
	link := scrappertypes.LinkResponse{URL: u}

	chat.Links = append(chat.Links, link)

	if s.IsURLInAdded(chatID, u) != true {
		t.Errorf("Incorrect value, got: %v; expected %v", s.IsURLInAdded(chatID, u), true)
	}
}

func TestAppendLinkToLinks_HappyPath(t *testing.T) {
	type TestCase struct {
		name   string
		chatID int64
		link   scrappertypes.LinkResponse
	}

	testCases := []TestCase{
		{
			name:   "chat registered, no errors expected, link must be added",
			chatID: int64(12345),
			link: scrappertypes.LinkResponse{ID: 1,
				URL:     "its.url",
				Tags:    []string{"work", "hobby"},
				Filters: []string{"sister", "wife"},
			},
		},
		{
			name:   "chat was not registered, no errors expected, link must be non-added",
			chatID: int64(12346),
			link: scrappertypes.LinkResponse{ID: 2,
				URL:     "its.another.url",
				Tags:    []string{"work2", "hobby2"},
				Filters: []string{"sister2", "wife2"},
			},
		},
	}

	s := &scrapper.Server{
		Chats:     make(map[int64]*scrappertypes.Chat),
		ChatMutex: sync.Mutex{},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(testCase.chatID, 10), http.NoBody)
			w := httptest.NewRecorder()

			s.RegisterChat(w, req)

			err := s.AppendLinkToLinks(testCase.chatID, &testCase.link)
			if err != nil {
				t.Errorf("Incorrect answer, got: %v; expected %v in case %v", err, nil, testCase.name)
			}

			s.ChatMutex.Lock()
			chat, exists := s.Chats[testCase.chatID]
			s.ChatMutex.Unlock()
			assert.True(t, exists, "Chat must exist")

			var found bool

			for _, link := range chat.Links {
				if link.URL == testCase.link.URL {
					assert.ElementsMatch(t, link.Tags, testCase.link.Tags, "Tags do not match")
					assert.ElementsMatch(t, link.Filters, testCase.link.Filters, "Filters do not match")

					found = true

					break
				}
			}

			assert.True(t, found, "Expected link was not found in chat")
		})
	}
}

func TestAppendLinkToLinks_ChatNotFound(t *testing.T) {
	type TestCase struct {
		name   string
		chatID int64
		link   *scrappertypes.LinkResponse
	}

	testCases := []TestCase{
		{
			name:   "chat registered, no errors expected, link must be added",
			chatID: int64(12345),
			link: &scrappertypes.LinkResponse{ID: 1,
				URL:     "its.url",
				Tags:    []string{"work", "hobby"},
				Filters: []string{"sister", "wife"},
			},
		},
		{
			name:   "chat was not registered, no errors expected, link must be non-added",
			chatID: int64(12346),
			link: &scrappertypes.LinkResponse{ID: 2,
				URL:     "its.another.url",
				Tags:    []string{"work2", "hobby2"},
				Filters: []string{"sister2", "wife2"},
			},
		},
	}

	s := &scrapper.Server{
		Chats:     make(map[int64]*scrappertypes.Chat),
		ChatMutex: sync.Mutex{},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			err := s.AppendLinkToLinks(testCase.chatID, testCase.link)
			if !errors.Is(err, e.ErrChatNotFound) {
				t.Errorf("Incorrect answer, got: %v; expected %v in case %v", err, e.ErrChatNotFound, testCase.name)
			}
		})
	}
}

func TestTwiceAppendLinkToLinks(t *testing.T) {
	type TestCase struct {
		name      string
		chatID    int64
		listLinks *scrappertypes.ListLinksResponse
	}

	testCases := []TestCase{
		{
			name:   "link to try add twice",
			chatID: int64(12345),
			listLinks: &scrappertypes.ListLinksResponse{Links: []scrappertypes.LinkResponse{
				{URL: "its.url"},
				{URL: "its.url"}}},
		},
		{
			name:   "second id with the same links to try add twice",
			chatID: int64(66666),
			listLinks: &scrappertypes.ListLinksResponse{Links: []scrappertypes.LinkResponse{
				{URL: "its.url"},
				{URL: "its.url"}}},
		},
	}

	s := &scrapper.Server{
		Chats:     make(map[int64]*scrappertypes.Chat),
		ChatMutex: sync.Mutex{},
	}

	for _, testCase := range testCases {
		req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(testCase.chatID, 10), http.NoBody)
		w := httptest.NewRecorder()

		s.RegisterChat(w, req)

		err := s.AppendLinkToLinks(testCase.chatID, &testCase.listLinks.Links[0])
		if err != nil {
			t.Errorf("Incorrect error adding first time, got: %v; expected %v with url:  %v", err,
				nil, &testCase.listLinks.Links[0].URL)
		}

		err = s.AppendLinkToLinks(testCase.chatID, &testCase.listLinks.Links[1])
		if !errors.Is(err, e.ErrLinkAlreadyExists) {
			t.Errorf("Incorrect error adding first time, got: %v; expected %v with url:  %v", err,
				e.ErrLinkAlreadyExists, &testCase.listLinks.Links[1].URL)
		}
	}
}

func TestRemoveLink(t *testing.T) {
	type TestCase struct {
		name             string
		indexToBeDeleted int
		given            []scrappertypes.LinkResponse
		expected         []scrappertypes.LinkResponse
	}

	listLinks := []scrappertypes.LinkResponse{
		{URL: "link1", Tags: []string{"tag1", "tag2"}, Filters: []string{"filter1", "filter2"}},
		{URL: "link2", Tags: []string{"tag3", "tag4"}, Filters: []string{"filter3", "filter4"}},
		{URL: "link3", Tags: []string{"tag5", "tag6"}, Filters: []string{"filter5", "filter6"}},
	}

	testCases := []TestCase{
		{
			name:             "deleting from the beginning of list",
			indexToBeDeleted: 0,
			given:            append([]scrappertypes.LinkResponse(nil), listLinks...),
			expected:         append([]scrappertypes.LinkResponse(nil), listLinks[1:]...),
		},
		{
			name:             "deleting from the middle of list",
			indexToBeDeleted: 1,
			given:            append([]scrappertypes.LinkResponse(nil), listLinks...),
			expected:         append([]scrappertypes.LinkResponse(nil), append(listLinks[:1], listLinks[2:]...)...),
		},
		{
			name:             "deleting from the end of list",
			indexToBeDeleted: 2,
			given:            append([]scrappertypes.LinkResponse(nil), listLinks...),
			expected:         append([]scrappertypes.LinkResponse(nil), listLinks[:2]...),
		},
	}

	for _, testCase := range testCases {
		server := &scrapper.Server{
			Chats:     make(map[int64]*scrappertypes.Chat),
			ChatMutex: sync.Mutex{},
			BotClient: new(scrapper.MockBotClient),
		}

		chatID := int64(12345)
		server.Chats[chatID] = &scrappertypes.Chat{}
		server.Chats[chatID].Links = testCase.given

		delReq := scrappertypes.RemoveLinkRequest{Link: testCase.given[testCase.indexToBeDeleted].URL}
		body, _ := json.Marshal(delReq)

		q := url.Values{}

		q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

		req := httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBuffer(body))
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		server.RemoveLink(w, req)

		resp := w.Result()

		err := resp.Body.Close()
		if err != nil {
			t.Errorf("Some error while testing: %v", e.ErrCloseBody)
		}

		for i, elem := range testCase.expected {
			if elem.URL != server.Chats[chatID].Links[i].URL {
				t.Errorf("Incorrect deleting, different values in index: %d; got: %v; expected %v",
					testCase.indexToBeDeleted, server.Chats[chatID].Links[i].URL, elem.URL)
			}
		}
	}
}
