package processing

import (
	"encoding/json"
	"fmt"
	"go-progira/internal/application/bot/clients"
	botmessages "go-progira/internal/domain/bot_messages"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	telegramtypes "go-progira/internal/domain/types/telegram_types"
	"log/slog"
	"net/url"
	"strings"
)

type StateChange func(id int, text string)

type State uint8

const (
	StateAwaitingStart State = iota
	StateStart
	stateAwaitingTagsForTrack
	stateAwaitingFiltersForTrack
)

type Manager struct {
	TgClient    clients.HTTPTelegramClient
	ScrapClient clients.HTTPScrapperClient
	States      map[int]State
	handlers    map[State]StateChange
	addRequests map[int]*scrappertypes.AddLinkRequest
}

func NewManager(tgClient clients.HTTPTelegramClient, scrapClient clients.HTTPScrapperClient) *Manager {
	return &Manager{
		tgClient,
		scrapClient,
		make(map[int]State),
		make(map[State]StateChange),
		make(map[int]*scrappertypes.AddLinkRequest),
	}
}

func (m Manager) HandleAwaitingStart(id int, text string) {
	parts := strings.Fields(text)

	switch parts[0] {
	case "/start":
		m.States[id] = StateStart

		m.ScrapClient.RegisterChat(int64(id))

		err := m.TgClient.SendMessage(id, botmessages.MsgHello)
		if err != nil {
			return
		}

		commands := []telegramtypes.BotCommand{
			{Command: "/track", Description: "Начать отслеживать ссылку"},
			{Command: "/untrack", Description: "Перестать отслеживать ссылку"},
			{Command: "/list", Description: "Показать отслеживаемые ссылки"},
			{Command: "/help", Description: "Справка"},
		}

		errSet := m.TgClient.SetBotCommands(commands)
		if errSet != nil {
			slog.Error("Setting bot commands wasn't worked successfully")
			slog.String("error", errSet.Error())

			return
		}
	case "/help":
		m.SendHelp(id)
	default:
		err := m.TgClient.SendMessage(id, botmessages.MsgUnknownCommand)
		if err != nil {
			return
		}
	}
}

func isValidURL(str string) bool {
	_, err := url.ParseRequestURI(str)
	return err == nil
}

func (m Manager) processListCommand(id int) {
	links, err := m.ScrapClient.GetLinks(int64(id))
	if err != nil {
		slog.Error("Error getting links",
			slog.String("error", err.Error()))
		return
	}

	if len(links.Links) == 0 {
		err = m.TgClient.SendMessage(id, botmessages.MsgNoSavedPages)
		if err != nil {
			slog.Error("Error sending message",
				slog.String("error", err.Error()))
		}
	} else {
		linkList := MakeLinkList(links.Links)

		err := m.TgClient.SendMessage(id, linkList)
		if err != nil {
			slog.Error("Error sending message",
				slog.String("error", err.Error()))
		}
	}
}

func (m Manager) handleStart(id int, text string) {
	parts := strings.Fields(text)

	switch parts[0] {
	case "/track":
		if len(parts) == 1 || !isValidURL(parts[1]) {
			err := m.TgClient.SendMessage(id, botmessages.MsgUnknownCommand)
			if err != nil {
				slog.Error("Error sending message",
					slog.String("error", err.Error()))
			}

			return
		}

		m.addRequests[id] = &scrappertypes.AddLinkRequest{Link: parts[1]}

		err := m.TgClient.SendMessage(id, botmessages.MsgAddTags)
		if err != nil {
			slog.Error("Error sending message",
				slog.String("error", err.Error()))

			return
		}

		m.States[id] = stateAwaitingTagsForTrack

	case "/untrack":
		if len(parts) == 1 || !isValidURL(parts[1]) {
			err := m.TgClient.SendMessage(id, botmessages.MsgUnknownCommand)
			if err != nil {
				slog.Error("Error sending message",
					slog.String("error", err.Error()))
			}

			return
		}

		delReq := scrappertypes.RemoveLinkRequest{Link: parts[1]}

		_, err := m.ScrapClient.RemoveLink(int64(id), delReq)
		if err != nil {
			slog.Error("Error removing link",
				slog.String("error", err.Error()),
				slog.String("link", parts[1]))

			return
		}

		errSendMes := m.TgClient.SendMessage(id, botmessages.MsgDeleted)
		if errSendMes != nil {
			slog.Error("Error sending message",
				slog.String("error", errSendMes.Error()))
		}

	case "/list":
		m.processListCommand(id)
	case "/help":
		m.SendHelp(id)

	default:
		err := m.TgClient.SendMessage(id, botmessages.MsgUnknownCommand)
		if err != nil {
			slog.Error("Error sending message",
				slog.String("error", err.Error()))
		}
	}
}

func MakeLinkList(links []scrappertypes.LinkResponse) string {
	var linksToSend strings.Builder

	for _, linkResp := range links {
		tagString := ""
		filterString := ""

		if len(linkResp.Tags) != 0 {
			tagString = fmt.Sprintf("Tags: %s\n", strings.Join(linkResp.Tags, ", "))
		}

		if len(linkResp.Filters) != 0 {
			filterString = fmt.Sprintf("Filters: %s\n", strings.Join(linkResp.Filters, ", "))
		}

		rec := fmt.Sprintf("Link: %s\n%s%s", linkResp.URL, tagString, filterString)
		linksToSend.WriteString(rec)
	}

	return linksToSend.String()
}

func (m Manager) SendHelp(id int) {
	err := m.TgClient.SendMessage(id, botmessages.MsgHelp)
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

	err := m.TgClient.SendMessage(id, botmessages.MsgAddFilters)
	if err != nil {
		return
	}

	m.States[id] = stateAwaitingFiltersForTrack
}

func (m Manager) handleAwaitingFiltersForTrack(id int, text string) {
	filters := splitByWords(text)
	m.addRequests[id].Filters = filters
	m.States[id] = StateStart
	_, err := m.ScrapClient.AddLink(int64(id), *m.addRequests[id])

	errSendMes := m.TgClient.SendMessage(id, botmessages.MsgSaved)
	if errSendMes != nil {
		return
	}

	if err != nil {
		return
	}
}

func (m Manager) buildHandlers() {
	m.handlers[StateAwaitingStart] = m.HandleAwaitingStart
	m.handlers[StateStart] = m.handleStart
	m.handlers[stateAwaitingTagsForTrack] = m.handleAwaitingTagsForTrack
	m.handlers[stateAwaitingFiltersForTrack] = m.handleAwaitingFiltersForTrack
}

func (m Manager) getUserState(id int) State {
	if _, ok := m.States[id]; !ok {
		m.States[id] = StateAwaitingStart
	}

	return m.States[id]
}

func (m Manager) Start() {
	commands := []telegramtypes.BotCommand{
		{Command: "/start", Description: "Запуск бота"},
		{Command: "/help", Description: "Справка"},
	}

	err := m.TgClient.SetBotCommands(commands)
	if err != nil {
		slog.Error("Setting bot commands wasn't worked successfully")
		slog.String("error", err.Error())
	}

	m.buildHandlers()

	var offset int

	for {
		data, _ := m.TgClient.Updates(offset, 1)
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
