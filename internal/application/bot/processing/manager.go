package processing

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/application/bot/clients"
	botmessages "go-progira/internal/domain/bot_messages"
	telegramtypes "go-progira/internal/domain/types/telegram_types"
	"strings"
	"time"
)

type StateChange func(id int, text string)

type state uint8

const (
	stateAwaitingStart state = iota
	stateStart
	stateListing
	stateAwaitingTagsForTrack
	stateAwaitingFiltersForTrack
	stateAwaitingTagsForUntrack
	stateAwaitingFiltersForUntrack
)

type Manager struct {
	tgClient    *clients.TelegramClient
	scrapClient *clients.ScrapperClient
	states      map[int]state
	handlers    map[state]StateChange
}

func NewManager(tgClient clients.TelegramClient, scrapClient clients.ScrapperClient) *Manager {
	return &Manager{
		&tgClient,
		&scrapClient,
		make(map[int]state),
		make(map[state]StateChange),
	}
}

func (m Manager) handleAwaitingStart(id int, text string) {
	parts := strings.Fields(text)
	if len(parts) == 1 && parts[0] == "/start" {
		m.states[id] = stateStart

		err := m.tgClient.SendMessage(id, botmessages.MsgHello)
		if err != nil {
			return
		}

	} else {
		err := m.tgClient.SendMessage(id, botmessages.MsgUnknownCommand)
		if err != nil {
			return
		}
		// неизвестная команда
		// вывести хелп
	}
}

func (m Manager) handleStart(id int, text string) {
	parts := strings.Fields(text)
	_ = m.tgClient.SendMessage(id, "Start!")

	switch parts[0] {
	case "/track":
		m.states[id] = stateAwaitingTagsForTrack
		// послать скрапперу вместе с ссылкой, проверить что ссылка корректный юрл

	case "/untrack":
		m.states[id] = stateAwaitingTagsForUntrack
		// послать скрапперу вместе с ссылкой, проверить что ссылка корректный юрл
	case "/list":
		m.states[id] = stateListing
	// послать скрапперу
	default:
		err := m.tgClient.SendMessage(id, botmessages.MsgUnknownCommand)
		if err != nil {
			return
		}
	}
}

func (m Manager) handleHelp(id int) {
	err := m.tgClient.SendMessage(id, botmessages.MsgHelp)
	if err != nil {
		return
	}

	m.states[id] = stateStart
}

func (m Manager) handleListing(id int, text string) {

}

//func getTags() {}

func (m Manager) handleAwaitingTagsForTrack(id int, text string) {}

//func getFilters() {}

func (m Manager) handleAwaitingFiltersForTrack(id int, text string) {}

func (m Manager) handleAwaitingTagsForUntrack(id int, text string) {}

func (m Manager) handleAwaitingFiltersForUntrack(id int, text string) {}

func (m *Manager) buildHandlers() {
	m.handlers[stateAwaitingStart] = m.handleAwaitingStart
	m.handlers[stateStart] = m.handleStart
	m.handlers[stateListing] = m.handleListing
	m.handlers[stateAwaitingTagsForTrack] = m.handleAwaitingTagsForTrack
	m.handlers[stateAwaitingFiltersForTrack] = m.handleAwaitingFiltersForTrack
	m.handlers[stateAwaitingTagsForUntrack] = m.handleAwaitingTagsForUntrack
	m.handlers[stateAwaitingFiltersForUntrack] = m.handleAwaitingFiltersForUntrack
}

func (m *Manager) getUserState(id int) state {
	if _, ok := m.states[id]; !ok {
		m.states[id] = stateAwaitingStart
	}

	return m.states[id]
}

func (m Manager) Start() {
	m.buildHandlers()
	for {
		data, _ := m.tgClient.Updates(1, 5)
		upds := telegramtypes.UpdatesResponse{}

		if err := json.Unmarshal(data, &upds); err != nil {
			return
		}
		fmt.Println(upds.Result[0].Message.Text)

		for i, _ := range upds.Result {
			if upds.Result != nil {
				id := upds.Result[i].Message.Chat.ID
				state := m.getUserState(id)
				m.handlers[state](id, upds.Result[i].Message.Text)
			}
		}

		time.Sleep(30 * time.Second)
	}
}
