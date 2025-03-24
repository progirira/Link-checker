package clients

import (
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	telegramtypes "go-progira/internal/domain/types/telegram_types"

	"github.com/stretchr/testify/mock"
)

type MockTgClient struct {
	mock.Mock
}

func (m *MockTgClient) Updates(offset, limit int) ([]byte, error) {
	args := m.Called(offset, limit)

	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTgClient) SendMessage(id int, msg string) error {
	args := m.Called(id, msg)

	return args.Error(0)
}

func (m *MockTgClient) SetBotCommands(commands []telegramtypes.BotCommand) error {
	args := m.Called(commands)

	return args.Error(0)
}

type MockScrapClient struct {
	mock.Mock
}

func (m *MockScrapClient) RegisterChat(chatID int64) {
	m.Called(chatID)
}

func (m *MockScrapClient) DeleteChat(chatID int64) {
	m.Called(chatID)
}

func (m *MockScrapClient) AddLink(chatID int64, request scrappertypes.AddLinkRequest) (*scrappertypes.LinkResponse, error) {
	args := m.Called(chatID, request)

	return args.Get(0).(*scrappertypes.LinkResponse), args.Error(1)
}

func (m *MockScrapClient) GetLinks(chatID int64) (*scrappertypes.ListLinksResponse, error) {
	args := m.Called(chatID)

	return args.Get(0).(*scrappertypes.ListLinksResponse), args.Error(1)
}

func (m *MockScrapClient) RemoveLink(chatID int64, request scrappertypes.RemoveLinkRequest) (*scrappertypes.LinkResponse, error) {
	args := m.Called(chatID, request)

	return args.Get(0).(*scrappertypes.LinkResponse), args.Error(1)
}
