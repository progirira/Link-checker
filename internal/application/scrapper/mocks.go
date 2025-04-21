package scrapper

import (
	"go-progira/internal/domain/types/bottypes"

	"github.com/stretchr/testify/mock"
)

type MockBotClient struct {
	mock.Mock
}

func (m *MockBotClient) SendUpdate(update bottypes.LinkUpdate) (err error) {
	args := m.Called(update)

	return args.Error(0)
}
