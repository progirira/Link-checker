package storage

import (
	"go-progira/internal/application/scrapper/api"
	botmessages "go-progira/internal/domain/bot_messages"
	bottypes "go-progira/internal/domain/types/bot_types"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"go-progira/pkg/e"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

type Storage interface {
	CreateChat(id int64) error
	DeleteChat(id int64) error
	AddLink(id int64, url string, tags, filters []string, content string) error
	RemoveLink(id int64, link string) error
	GetLinks(id int64) ([]scrappertypes.LinkResponse, error)
	IsURLInAdded(id int64, u string) bool
	Update() []bottypes.LinkUpdate
}

type DictionaryStorage struct {
	mutex sync.RWMutex
	Chats map[int64]*scrappertypes.Chat
}

func (d *DictionaryStorage) CreateChat(id int64) error {
	d.mutex.Lock()

	defer d.mutex.Unlock()

	if _, exists := d.Chats[id]; exists {
		slog.Error(e.ErrChatAlreadyExists.Error())

		return e.ErrChatAlreadyExists
	}

	d.Chats[id] = &scrappertypes.Chat{ID: id, Links: []scrappertypes.LinkResponse{}}

	return nil
}

func (d *DictionaryStorage) DeleteChat(id int64) error {
	d.mutex.Lock()

	defer d.mutex.Unlock()

	if _, exists := d.Chats[id]; !exists {
		slog.Error(e.ErrChatNotFound.Error())

		return e.ErrChatNotFound
	}

	delete(d.Chats, id)

	return nil
}

func (d *DictionaryStorage) GetLinks(id int64) ([]scrappertypes.LinkResponse, error) {
	d.mutex.RLock()

	defer d.mutex.RUnlock()

	chat, exists := d.Chats[id]
	if !exists {
		slog.Error(e.ErrChatNotFound.Error())
		slog.String("id", strconv.FormatInt(id, 10))

		return nil, e.ErrChatNotFound
	}

	return chat.Links, nil
}

func (d *DictionaryStorage) AddLink(id int64, url string, tags, filters []string, content string) error {
	d.mutex.RLock()

	_, exists := d.Chats[id]

	d.mutex.RUnlock()

	if !exists {
		return e.ErrChatNotFound
	}

	link := &scrappertypes.LinkResponse{
		ID:          int64(len(d.Chats[id].Links) + 1),
		URL:         url,
		Tags:        tags,
		Filters:     filters,
		LastVersion: content,
		LastChecked: time.Now(),
	}

	return d.AppendLinkToLinks(id, link)
}

func (d *DictionaryStorage) RemoveLink(id int64, link string) error {
	d.mutex.Lock()

	defer d.mutex.Unlock()

	chat, exists := d.Chats[id]

	if !exists {
		slog.Error(e.ErrChatNotFound.Error())
		return e.ErrChatNotFound
	}

	for i, chatLink := range chat.Links {
		if chatLink.URL == link {
			chat.Links = append(chat.Links[:i], chat.Links[i+1:]...)

			return nil
		}
	}

	slog.Debug(e.ErrLinkNotFound.Error())
	slog.String("link", link)

	return e.ErrLinkNotFound
}

func (d *DictionaryStorage) AppendLinkToLinks(chatID int64, link *scrappertypes.LinkResponse) error {
	d.mutex.RLock()

	chat, exists := d.Chats[chatID]

	d.mutex.RUnlock()

	if !exists {
		return e.ErrChatNotFound
	}

	if !d.IsURLInAdded(chatID, link.URL) {
		d.mutex.RLock()

		chat.Links = append(chat.Links, *link)

		d.mutex.RUnlock()
	} else {
		return e.ErrLinkAlreadyExists
	}

	return nil
}

func (d *DictionaryStorage) IsURLInAdded(id int64, u string) bool {
	d.mutex.RLock()

	defer d.mutex.RUnlock()

	if len(d.Chats[id].Links) == 0 {
		return false
	}

	for _, l := range d.Chats[id].Links {
		if l.URL == u {
			return true
		}
	}

	return false
}

func (d *DictionaryStorage) GetAllIDs(id int64) ([]scrappertypes.LinkResponse, error) {
	d.mutex.RLock()

	defer d.mutex.RUnlock()

	chat, exists := d.Chats[id]
	if !exists {
		slog.Error(e.ErrChatNotFound.Error())
		slog.String("id", strconv.FormatInt(id, 10))

		return nil, e.ErrChatNotFound
	}

	return chat.Links, nil
}

func (d *DictionaryStorage) Update() []bottypes.LinkUpdate {
	var updates []bottypes.LinkUpdate

	var currentVersion string

	var err error

	d.mutex.RLock()

	for chatID, chat := range d.Chats {
		for _, link := range chat.Links {
			if updater, ok := api.GetUpdater(link.URL); ok {
				currentVersion, err = updater(link.URL)
				if err != nil {
					slog.Error(
						"Updater error",
						slog.String("url", link.URL),
					)

					continue
				}
			} else {
				slog.Error(
					e.ErrWrongURLFormat.Error(),
					slog.String("url", link.URL),
				)

				continue
			}

			if currentVersion != "" {
				d.mutex.RUnlock()
				update := d.updateLinkContent(chatID, link.URL, currentVersion)

				if update.URL != "" {
					updates = append(updates, update)
				}

				d.mutex.RLock()
			}
		}
	}

	d.mutex.RUnlock()

	return updates
}

func (d *DictionaryStorage) updateLinkContent(id int64, url, lastContent string) bottypes.LinkUpdate {
	var update bottypes.LinkUpdate

	d.mutex.Lock()
	for i, l := range d.Chats[id].Links {
		if url == l.URL {
			if lastContent != "" && lastContent != l.LastVersion {
				d.Chats[id].Links[i].LastVersion = lastContent
				d.Chats[id].Links[i].LastChecked = time.Now()

				update = bottypes.LinkUpdate{
					ID:          id,
					URL:         l.URL,
					Description: botmessages.MsgUpdatesHappened,
					TgChatIDs:   []int64{id},
				}

				return update
			}
		}
	}

	d.mutex.Unlock()

	return bottypes.LinkUpdate{}
}
