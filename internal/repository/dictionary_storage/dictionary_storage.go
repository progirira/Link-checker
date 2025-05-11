package repository

import (
	"context"
	"go-progira/internal/domain/types/scrappertypes"
	"go-progira/pkg/e"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

type ChatStorage interface {
	CreateChat(ctx context.Context, id int64) error
	DeleteChat(ctx context.Context, id int64) error
}

type LinkStorage interface {
	AddLink(ctx context.Context, id int64, url string, tags, filters []string) error
	RemoveLink(ctx context.Context, id int64, link string) error
	GetLinks(ctx context.Context, id int64) ([]scrappertypes.LinkResponse, error)
	IsURLInAdded(ctx context.Context, id int64, u string) bool
	GetBatchOfLinks(context.Context, int, int64) ([]scrappertypes.LinkResponse, int64)
	DeleteTag(ctx context.Context, id int64, tag string) error
}

type UpdateStorage interface {
	GetPreviousUpdate(ctx context.Context, ID int64) time.Time
	SaveLastUpdate(ctx context.Context, ID int64, updTime time.Time) error
	GetTgChatIDsForLink(ctx context.Context, link string) []int64
}

type LinkService interface {
	ChatStorage
	LinkStorage
	UpdateStorage
}

type DictionaryStorage struct {
	mutex sync.RWMutex
	Chats map[int64]*scrappertypes.Chat
}

func (d *DictionaryStorage) CreateChat(_ context.Context, id int64) error {
	d.mutex.Lock()

	defer d.mutex.Unlock()

	if _, exists := d.Chats[id]; exists {
		slog.Error(e.ErrChatAlreadyExists.Error())

		return e.ErrChatAlreadyExists
	}

	d.Chats[id] = &scrappertypes.Chat{ID: id, Links: []scrappertypes.LinkResponse{}}

	return nil
}

func (d *DictionaryStorage) DeleteChat(_ context.Context, id int64) error {
	d.mutex.Lock()

	defer d.mutex.Unlock()

	if _, exists := d.Chats[id]; !exists {
		slog.Error(e.ErrChatNotFound.Error())

		return e.ErrChatNotFound
	}

	delete(d.Chats, id)

	return nil
}

func (d *DictionaryStorage) GetLinks(_ context.Context, id int64) ([]scrappertypes.LinkResponse, error) {
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

func (d *DictionaryStorage) AddLink(ctx context.Context, id int64, url string, tags, filters []string, content string) error {
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

	return d.AppendLinkToLinks(ctx, id, link)
}

func (d *DictionaryStorage) RemoveLink(_ context.Context, id int64, link string) error {
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

func (d *DictionaryStorage) AppendLinkToLinks(ctx context.Context, chatID int64, link *scrappertypes.LinkResponse) error {
	d.mutex.RLock()

	chat, exists := d.Chats[chatID]

	d.mutex.RUnlock()

	if !exists {
		return e.ErrChatNotFound
	}

	if !d.IsURLInAdded(ctx, chatID, link.URL) {
		d.mutex.RLock()

		chat.Links = append(chat.Links, *link)

		d.mutex.RUnlock()
	} else {
		return e.ErrLinkAlreadyExists
	}

	return nil
}

func (d *DictionaryStorage) IsURLInAdded(_ context.Context, id int64, u string) bool {
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

func (d *DictionaryStorage) GetAllIDs(_ context.Context, id int64) ([]scrappertypes.LinkResponse, error) {
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

func (d *DictionaryStorage) GetBatchOfLinks(_ context.Context, batch int, lastID int64) (links []scrappertypes.LinkResponse,
	newLastID int64) {
	newLastID = lastID

	d.mutex.Lock()

	for id, chat := range d.Chats {
		if id <= lastID {
			continue
		}

		links = append(links, chat.Links...)
		newLastID = id

		if len(links) >= batch {
			return links, newLastID
		}
	}

	d.mutex.Unlock()

	return links, newLastID
}

// func (d *DictionaryStorage) GetPreviousUpdate(ctx context.Context, ID int64) time.Time {}
//
// func (d *DictionaryStorage) SaveLastUpdate(ctx context.Context, ID int64, updTime time.Time) error
//
// func (d *DictionaryStorage) GetTgChatIDsForLink(ctx context.Context, link string) []int64
