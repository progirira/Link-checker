package processing_test

import (
	"go-progira/internal/application/bot/clients"
	"go-progira/internal/application/bot/processing"
	"go-progira/internal/domain/botmessages"
	"go-progira/internal/domain/types/scrappertypes"
	"go-progira/internal/domain/types/telegramtypes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandleAwaitingStart_Start(t *testing.T) {
	type TestCase struct {
		name     string
		chatID   int
		command  string
		expected string
	}

	testCases := []TestCase{
		{
			name:     "command start",
			chatID:   12346,
			command:  "/start",
			expected: botmessages.MsgHello,
		},
	}

	mockTg := new(clients.MockTgClient)
	mockScrap := new(clients.MockScrapClient)

	manager := processing.Manager{
		States:      make(map[int]processing.State),
		ScrapClient: mockScrap,
		TgClient:    mockTg,
	}

	for _, testCase := range testCases {
		assert.Equal(t, processing.StateAwaitingStart, manager.States[testCase.chatID])
		mockScrap.On("RegisterChat", int64(testCase.chatID)).Return()
		mockTg.On("SendMessage", testCase.chatID, botmessages.MsgHello).Return(nil)

		commands := []telegramtypes.BotCommand{
			{Command: "/track", Description: "Начать отслеживать ссылку"},
			{Command: "/untrack", Description: "Перестать отслеживать ссылку"},
			{Command: "/list", Description: "Показать отслеживаемые ссылки"},
			{Command: "/help", Description: "Справка"},
		}
		mockTg.On("SetBotCommands", commands).Return(nil)

		manager.HandleAwaitingStart(testCase.chatID, "/start")

		assert.Equal(t, processing.StateStart, manager.States[testCase.chatID])

		mockScrap.AssertCalled(t, "RegisterChat", int64(testCase.chatID))
		mockTg.AssertCalled(t, "SendMessage", testCase.chatID, botmessages.MsgHello)
	}
}

func TestHandleAwaitingStart_HelpAndUnknown(t *testing.T) {
	type TestCase struct {
		name     string
		chatID   int
		command  string
		expected string
	}

	testCases := []TestCase{
		{
			name:     "command help",
			chatID:   12347,
			command:  "/help",
			expected: botmessages.MsgHelp,
		},
		{
			name:     "unknown command",
			chatID:   12345,
			command:  "jump",
			expected: botmessages.MsgUnknownCommand,
		},
	}

	mockTg := new(clients.MockTgClient)
	mockScrap := new(clients.MockScrapClient)

	manager := processing.Manager{
		States:      make(map[int]processing.State),
		ScrapClient: mockScrap,
		TgClient:    mockTg,
	}

	for _, testCase := range testCases {
		assert.Equal(t, processing.StateAwaitingStart, manager.States[testCase.chatID])
		mockTg.On("SendMessage", testCase.chatID, testCase.expected).Return(nil)
		manager.HandleAwaitingStart(testCase.chatID, testCase.command)

		mockTg.AssertCalled(t, "SendMessage", testCase.chatID, testCase.expected)
	}
}

func TestMakeLinkList(t *testing.T) {
	type TestCase struct {
		name     string
		linkList []scrappertypes.LinkResponse
		expected string
	}

	testCases := []TestCase{
		{
			name: "tags and filters are in link",
			linkList: []scrappertypes.LinkResponse{
				{URL: "link",
					Tags:    []string{"tag1", "tag2", "tag3"},
					Filters: []string{"filter1", "filter2", "filter3"},
				},
			},
			expected: "Link: link\nTags: tag1, tag2, tag3\nFilters: filter1, filter2, filter3\n",
		},
		{
			name: "only tags are in link",
			linkList: []scrappertypes.LinkResponse{
				{URL: "link",
					Tags:    []string{"tag1", "tag2", "tag3"},
					Filters: []string{},
				},
			},
			expected: "Link: link\nTags: tag1, tag2, tag3\n",
		},
		{
			name: "only filters are in link",
			linkList: []scrappertypes.LinkResponse{
				{URL: "link",
					Tags:    []string{},
					Filters: []string{"filter1", "filter2", "filter3"},
				},
			},
			expected: "Link: link\nFilters: filter1, filter2, filter3\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(tt *testing.T) {
			tt.Parallel()

			got := processing.MakeLinkList(testCase.linkList)
			if got != testCase.expected {
				t.Errorf("Incorrect value, got: %v; expected %v", got, testCase.expected)
			}
		})
	}
}
