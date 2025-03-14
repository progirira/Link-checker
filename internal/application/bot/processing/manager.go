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
)

type StateChange func(id int, text string)

type state uint8

const (
	stateAwaitingStart state = iota
	stateStart
	stateAwaitingTagsForTrack
	stateAwaitingFiltersForTrack
)

type Manager struct {
	tgClient    *clients.TelegramClient
	scrapClient *clients.ScrapperClient
	states      map[int]state
	handlers    map[state]StateChange
	addRequests map[int]*scrappertypes.AddLinkRequest
}

func NewManager(tgClient *clients.TelegramClient, scrapClient *clients.ScrapperClient) *Manager {
	return &Manager{
		tgClient,
		scrapClient,
		make(map[int]state),
		make(map[state]StateChange),
		make(map[int]*scrappertypes.AddLinkRequest),
	}
}

func (m Manager) handleAwaitingStart(id int, text string) {
	parts := strings.Fields(text)

	switch parts[0] {
	case "/start":
		m.states[id] = stateStart

		m.scrapClient.RegisterChat(int64(id))

		err := m.tgClient.SendMessage(id, botmessages.MsgHello)
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

func isValidURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	return err == nil
}

func (m Manager) handleStart(id int, text string) {
	parts := strings.Fields(text)

	switch parts[0] {
	case "/track":
		if isValidURL(parts[1]) {
			m.addRequests[id] = &scrappertypes.AddLinkRequest{Link: parts[1]}

			err := m.tgClient.SendMessage(id, botmessages.MsgAddTags)
			if err != nil {
				return
			}

			m.states[id] = stateAwaitingTagsForTrack
		}
	case "/untrack":
		if isValidURL(parts[1]) {
			delReq := scrappertypes.RemoveLinkRequest{Link: parts[1]}

			_, err := m.scrapClient.RemoveLink(int64(id), delReq)
			if err != nil {
				return
			}

			err = m.tgClient.SendMessage(id, botmessages.MsgAddTags)
			if err != nil {
				return
			}
		}
	case "/list":
		links, err := m.scrapClient.GetLinks(int64(id))
		if err != nil {
			return
		}

		if len(links.Links) == 0 {
			_ = m.tgClient.SendMessage(id, botmessages.MsgNoSavedPages)
		} else {
			linkList := makeLinkList(links.Links)

			err := m.tgClient.SendMessage(id, linkList)
			if err != nil {
				return
			}
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

func makeLinkList(links []scrappertypes.LinkResponse) string {
	var linksToSend strings.Builder

	for _, linkResp := range links {
		rec := fmt.Sprintf("%s Tags: %s Filters: %s", linkResp.URL,
			strings.Join(linkResp.Tags, ""), strings.Join(linkResp.Filters, ""))
		linksToSend.WriteString(rec)
		linksToSend.WriteString("\n")
	}

	return linksToSend.String()
}

func (m Manager) sendHelp(id int) {
	err := m.tgClient.SendMessage(id, botmessages.MsgHelp)
	if err != nil {
		return
	}
}

func splitByWords(text string) []string {
	return strings.Fields(text)
}

func (m Manager) handleAwaitingTagsForTrack(id int, text string) {
	tags := splitByWords(text)
	m.addRequests[id].Tags = tags

	err := m.tgClient.SendMessage(id, botmessages.MsgAddFilters)
	if err != nil {
		return
	}

	m.states[id] = stateAwaitingFiltersForTrack
}

func (m Manager) handleAwaitingFiltersForTrack(id int, text string) {
	filters := splitByWords(text)
	m.addRequests[id].Filters = filters
	m.states[id] = stateStart
	_, err := m.scrapClient.AddLink(int64(id), *m.addRequests[id])

	errSendMes := m.tgClient.SendMessage(id, botmessages.MsgSaved)
	if errSendMes != nil {
		return
	}

	if err != nil {
		return
	}
}

func (m *Manager) buildHandlers() {
	m.handlers[stateAwaitingStart] = m.handleAwaitingStart
	m.handlers[stateStart] = m.handleStart
	m.handlers[stateAwaitingTagsForTrack] = m.handleAwaitingTagsForTrack
	m.handlers[stateAwaitingFiltersForTrack] = m.handleAwaitingFiltersForTrack
}

func (m *Manager) getUserState(id int) state {
	if _, ok := m.states[id]; !ok {
		m.states[id] = stateAwaitingStart
	}

	return m.states[id]
}

func (m Manager) Start() {
	m.buildHandlers()

	var offset int

	for {
		data, _ := m.tgClient.Updates(offset, 1)
		upds := telegramtypes.UpdatesResponse{}

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
	}
}
