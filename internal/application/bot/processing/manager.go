package processing

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/application/bot/clients"
	botmessages "go-progira/internal/domain/bot_messages"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	telegramtypes "go-progira/internal/domain/types/telegram_types"
	"net/url"
	"strings"
	"time"
)

type StateChange func(id int, text string)

type state uint8

const (
	stateAwaitingStart state = iota
	stateStart
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
	addRequests map[int]scrappertypes.AddLinkRequest
}

func NewManager(tgClient clients.TelegramClient, scrapClient clients.ScrapperClient) *Manager {
	return &Manager{
		&tgClient,
		&scrapClient,
		make(map[int]state),
		make(map[state]StateChange),
		make(map[int]scrappertypes.AddLinkRequest),
	}
}

func (m Manager) handleAwaitingStart(id int, text string) {
	parts := strings.Fields(text)
	if len(parts) == 1 && parts[0] == "/start" {
		m.states[id] = stateStart

		err := m.scrapClient.RegisterChat(int64(id))
		if err != nil {
			return
		}
		_ = m.tgClient.SendMessage(id, "Hello")
		err = m.tgClient.SendMessage(id, botmessages.MsgHello)
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

func isValidURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	return err == nil
}

func (m Manager) handleStart(id int, text string) {
	parts := strings.Fields(text)
	fmt.Println(parts[0])
	fmt.Println(m.getUserState(id))
	switch parts[0] {
	case "/track":
		m.states[id] = stateAwaitingTagsForTrack
		if isValidURL(parts[1]) {
			m.addRequests[id] = scrappertypes.AddLinkRequest{Link: parts[1]}
		}
	case "/untrack":
		m.states[id] = stateAwaitingTagsForUntrack
		if isValidURL(parts[1]) {
			delReq := scrappertypes.RemoveLinkRequest{Link: parts[1]}
			_, err := m.scrapClient.RemoveLink(int64(id), delReq)
			if err != nil {
				return
			}
		}
	case "/list":
		links, err := m.scrapClient.GetLinks(int64(id))
		if err != nil {
			return
		}
		var linksToSend strings.Builder
		for _, linkResp := range links.Links {
			rec := fmt.Sprintf("%s Tags: %s Filters: %s", linkResp.URL,
				linkResp.Tags, linkResp.Filters)
			linksToSend.WriteString("/n")
			linksToSend.WriteString(rec)
		}
		err = m.tgClient.SendMessage(id, linksToSend.String())
		if err != nil {
			return
		}

	case "/help":
		m.sendHelp(id)

	default:
		err := m.tgClient.SendMessage(id, botmessages.MsgUnknownCommand)
		if err != nil {
			return
		}
	}
}

func (m Manager) sendHelp(id int) {
	err := m.tgClient.SendMessage(id, botmessages.MsgHelp)
	if err != nil {
		return
	}
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
	m.buildHandlers() // Инициализация обработчиков
	var offset int

	for {
		data, _ := m.tgClient.Updates(offset, 1)
		upds := telegramtypes.UpdatesResponse{}

		fmt.Println("Data received:", string(data)) // Отладочный вывод получения данных
		if err := json.Unmarshal(data, &upds); err != nil {
			fmt.Println("Error while unmarshalling:", err)
			return
		}

		for _, res := range upds.Result {
			if res.Message != nil {
				id := res.Message.Chat.ID
				state := m.getUserState(id)

				m.handlers[state](id, res.Message.Text)

				offset = res.ID + 1
			}
		}

		time.Sleep(100 * time.Microsecond)
	}
}
