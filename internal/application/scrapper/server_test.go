package scrapper_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go-progira/internal/application/scrapper"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/internal/repository/storage"
	"go-progira/pkg/e"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterChat(t *testing.T) {
	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	s := &scrapper.Server{
		Storage: dict,
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

	_, err := s.Storage.GetLinks(chatID)

	if errors.Is(err, e.ErrChatNotFound) {
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
	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	s := &scrapper.Server{
		Storage: dict,
	}

	chatID := int64(12345)
	u := "url"
	req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(chatID, 10), http.NoBody)
	w := httptest.NewRecorder()

	s.RegisterChat(w, req)

	if s.Storage.IsURLInAdded(chatID, u) {
		t.Errorf("Incorrect value, got: %v; expected %v", s.Storage.IsURLInAdded(chatID, u), false)
	}

	errAdd := s.Storage.AddLink(chatID, u, []string{}, []string{}, "")
	if errAdd != nil {
		t.Errorf("Incorrect value, got: %v; expected %v", s.Storage.IsURLInAdded(chatID, u), true)
	}

	if !s.Storage.IsURLInAdded(chatID, u) {
		t.Errorf("Incorrect value, got: %v; expected %v", s.Storage.IsURLInAdded(chatID, u), true)
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

	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	s := &scrapper.Server{
		Storage: dict,
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(testCase.chatID, 10), http.NoBody)
			w := httptest.NewRecorder()

			s.RegisterChat(w, req)

			err := s.Storage.AddLink(testCase.chatID, testCase.link.URL, testCase.link.Tags, testCase.link.Filters, "")
			if err != nil {
				t.Errorf("Incorrect answer, got: %v; expected %v in case %v", err, nil, testCase.name)
			}

			var found bool

			links, errGet := s.Storage.GetLinks(testCase.chatID)
			if errGet != nil {
				t.Errorf("Incorrect answer, got: %v; expected %v in case %v", errGet, nil, testCase.name)
			}

			for _, link := range links {
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
			name:   "chat was not registered, link must be non-added",
			chatID: int64(12345),
			link: &scrappertypes.LinkResponse{ID: 1,
				URL:     "its.url",
				Tags:    []string{"work", "hobby"},
				Filters: []string{"sister", "wife"},
			},
		},
	}

	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	s := &scrapper.Server{
		Storage: dict,
	}

	for _, testCase := range testCases {
		err := s.Storage.AddLink(testCase.chatID, testCase.link.URL, testCase.link.Tags, testCase.link.Filters, "")
		if !errors.Is(err, e.ErrChatNotFound) {
			t.Errorf("Incorrect answer, got: %v; expected %v in case %v", err, e.ErrChatNotFound, testCase.name)
		}
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

	dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
	s := &scrapper.Server{
		Storage: dict,
	}

	for _, testCase := range testCases {
		req := httptest.NewRequest(http.MethodPost, "/tg-chat/"+strconv.FormatInt(testCase.chatID, 10), http.NoBody)
		w := httptest.NewRecorder()

		s.RegisterChat(w, req)

		err := s.Storage.AddLink(testCase.chatID, testCase.listLinks.Links[0].URL, []string{}, []string{}, "")
		if err != nil {
			t.Errorf("Incorrect error adding first time, got: %v; expected %v with url:  %v", err,
				nil, &testCase.listLinks.Links[0].URL)
		}

		err = s.Storage.AddLink(testCase.chatID, testCase.listLinks.Links[1].URL, []string{}, []string{}, "")
		if !errors.Is(err, e.ErrLinkAlreadyExists) {
			t.Errorf("Incorrect error adding second time, got: %v; expected %v with url:  %v", err,
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

	testCases := []TestCase{
		{
			name:             "deleting from the beginning of list",
			indexToBeDeleted: 0,
			given: []scrappertypes.LinkResponse{
				{URL: "link1", Tags: []string{"tag1", "tag2"}, Filters: []string{"filter1", "filter2"}},
				{URL: "link2", Tags: []string{"tag3", "tag4"}, Filters: []string{"filter3", "filter4"}},
				{URL: "link3", Tags: []string{"tag5", "tag6"}, Filters: []string{"filter5", "filter6"}},
			},
			expected: []scrappertypes.LinkResponse{
				{URL: "link2", Tags: []string{"tag3", "tag4"}, Filters: []string{"filter3", "filter4"}},
				{URL: "link3", Tags: []string{"tag5", "tag6"}, Filters: []string{"filter5", "filter6"}},
			},
		},
		{
			name:             "deleting from the middle of list",
			indexToBeDeleted: 1,
			given: []scrappertypes.LinkResponse{
				{URL: "link1", Tags: []string{"tag1", "tag2"}, Filters: []string{"filter1", "filter2"}},
				{URL: "link2", Tags: []string{"tag3", "tag4"}, Filters: []string{"filter3", "filter4"}},
				{URL: "link3", Tags: []string{"tag5", "tag6"}, Filters: []string{"filter5", "filter6"}},
			},
			expected: []scrappertypes.LinkResponse{
				{URL: "link1", Tags: []string{"tag1", "tag2"}, Filters: []string{"filter1", "filter2"}},
				{URL: "link3", Tags: []string{"tag5", "tag6"}, Filters: []string{"filter5", "filter6"}},
			},
		},
		{
			name:             "deleting from the end of list",
			indexToBeDeleted: 2,
			given: []scrappertypes.LinkResponse{
				{URL: "link1", Tags: []string{"tag1", "tag2"}, Filters: []string{"filter1", "filter2"}},
				{URL: "link2", Tags: []string{"tag3", "tag4"}, Filters: []string{"filter3", "filter4"}},
				{URL: "link3", Tags: []string{"tag5", "tag6"}, Filters: []string{"filter5", "filter6"}},
			},
			expected: []scrappertypes.LinkResponse{
				{URL: "link1", Tags: []string{"tag1", "tag2"}, Filters: []string{"filter1", "filter2"}},
				{URL: "link2", Tags: []string{"tag3", "tag4"}, Filters: []string{"filter3", "filter4"}},
			},
		},
	}

	for i, testCase := range testCases {
		dict := &storage.DictionaryStorage{Chats: make(map[int64]*scrappertypes.Chat)}
		server := &scrapper.Server{
			Storage:   dict,
			BotClient: new(scrapper.MockBotClient),
		}

		chatID := int64(12345 + i)

		err := server.Storage.CreateChat(chatID)
		if err != nil {
			t.Errorf("Incorrect error, got: %v; expected %v", err, nil)
		}

		for _, link := range testCase.given {
			errAdd := server.Storage.AddLink(chatID, link.URL, link.Tags, link.Filters, "")
			if errAdd != nil {
				t.Errorf("Incorrect error adding, got: %v; expected %v in case %v", errAdd, nil, testCase.name)
			}
		}

		delReq := scrappertypes.RemoveLinkRequest{Link: testCase.given[testCase.indexToBeDeleted].URL}
		body, _ := json.Marshal(delReq)

		q := url.Values{}

		q.Add("Tg-Chat-Id", fmt.Sprintf("%d", chatID))

		req := httptest.NewRequest(http.MethodDelete, "/links", bytes.NewBuffer(body))
		req.URL.RawQuery = q.Encode()
		w := httptest.NewRecorder()

		server.RemoveLink(w, req)

		resp := w.Result()

		err = resp.Body.Close()
		if err != nil {
			t.Errorf("Some error while testing: %v", e.ErrCloseBody)
		}

		links, errGet := server.Storage.GetLinks(chatID)
		if errGet != nil {
			t.Errorf("Incorrect answer, got: %v; expected %v in case %v", errGet, nil, testCase.name)
		}

		if len(links) != len(testCase.expected) {
			t.Errorf("Incorrect length, got: %v; expected %v", len(links), len(testCase.expected))
			return
		}

		for i, elem := range testCase.expected {
			if elem.URL != links[i].URL {
				t.Errorf("Incorrect deleting, different values in index: %d; got: %v; expected %v",
					testCase.indexToBeDeleted, links[i].URL, elem.URL)
			}
		}
	}
}
