package repository

import (
	"context"
	"github.com/jackc/pgx/v4/pgxpool"
	scrappertypes "go-progira/internal/domain/types/scrapper_types"
	"log/slog"
	"time"
)

type SQLStorage struct {
	db *pgxpool.Pool
}

func NewSQLStorage(dbURL string) (*SQLStorage, error) {
	var pool *pgxpool.Pool
	var err error

	pool, err = pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		slog.Error(ErrPoolCreate.Error(),
			slog.String("error", err.Error()),
			slog.String("database URL", dbURL))
		return nil, err
	}

	return &SQLStorage{db: pool}, nil
}

func (s *SQLStorage) CreateChat(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, "INSERT INTO users (telegram_id) VALUES ($1) ON CONFLICT (telegram_id) DO NOTHING", id)
	if err != nil {
		slog.Error(ErrCreateChat.Error(),
			slog.String("error", err.Error()))
	}

	return err
}

func (s *SQLStorage) DeleteChat(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, "DELETE FROM users WHERE telegram_id = $1", id)
	if err != nil {
		slog.Error(ErrDeleteChat.Error(),
			slog.String("error", err.Error()))
	}

	return err
}

func (s *SQLStorage) AddLink(ctx context.Context, id int64, url string, tags, filters []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	var linkID int64

	errQuery := tx.QueryRow(ctx, "INSERT INTO links (url, changed_at) VALUES ($1, NOW()) ON CONFLICT (url) DO NOTHING RETURNING id", url).Scan(&linkID)

	if errQuery != nil {
		slog.Error("Query Exec error")

		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return errRollback
		}

		return errQuery
	}

	if linkID == 0 {
		err = tx.QueryRow(ctx, "SELECT id FROM links WHERE url = $1", url).Scan(&linkID)
		if err != nil {
			slog.Error(ErrExecQuery.Error(),
				slog.String("error", err.Error()))

			slog.Error("Invalid ID",
				slog.Int("id", int(linkID)))
		}
	}

	_, err = tx.Exec(ctx, "INSERT INTO link_users (user_id, link_id) VALUES ((SELECT id FROM users WHERE telegram_id = $1), $2) ON CONFLICT DO NOTHING", id, linkID)
	if err != nil {
		slog.Error(ErrExecQuery.Error(),
			slog.String("error", err.Error()))
	}

	for _, tag := range tags {
		var tagID int64

		err := tx.QueryRow(ctx, "INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id", tag).Scan(&tagID)
		if err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", linkID, tagID)

			if err != nil {
				slog.Error(ErrExecQuery.Error())
				slog.String("error", err.Error())
			}
		}
	}

	for _, filter := range filters {
		var filterID int64

		err := tx.QueryRow(ctx, "INSERT INTO filters (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id", filter).Scan(&filterID)
		if err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO link_filters (link_id, filter_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", linkID, filterID)

			if err != nil {
				slog.Error(ErrExecQuery.Error())
				slog.String("error", err.Error())
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *SQLStorage) RemoveLink(ctx context.Context, id int64, link string) error {
	_, err := s.db.Exec(ctx, `
        DELETE FROM link_users 
        WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1) 
        AND link_id = (SELECT id FROM links WHERE url = $2)`, id, link)
	if err != nil {
		slog.Error(ErrRemoveLink.Error())
		slog.String("error", err.Error())
		slog.Int("id", int(id))
	}

	return err
}

func (s *SQLStorage) GetTags(ctx context.Context, id int64) map[int64][]string {
	rows, err := s.db.Query(ctx, `
        SELECT lt.link_id, t.name 
        FROM tags t
        JOIN link_tags lt ON lt.tag_id = t.id
        JOIN link_users lu ON lt.link_id = lu.link_id
        JOIN users u ON u.id = lu.user_id
        WHERE u.telegram_id = $1`, id)
	if err != nil {
		return nil
	}

	defer rows.Close()

	tagsByID := make(map[int64][]string)

	for rows.Next() {
		var linkID int64
		var tag string

		err := rows.Scan(&linkID, &tag)
		if err != nil {
			return nil
		}

		if _, ok := tagsByID[linkID]; !ok {
			tagsByID[linkID] = []string{}
		}

		tagsByID[linkID] = append(tagsByID[linkID], tag)
	}

	return tagsByID
}

func (s *SQLStorage) GetFilters(ctx context.Context, id int64) map[int64][]string {
	rows, err := s.db.Query(ctx, `
        SELECT lf.link_id, f.name 
        FROM filters f
        JOIN link_filters lf ON lf.filter_id = f.id
        JOIN link_users lu ON lf.link_id = lu.link_id
        JOIN users u ON u.id = lu.user_id
        WHERE u.telegram_id = $1`, id)
	if err != nil {
		return nil
	}

	defer rows.Close()

	filtersByID := make(map[int64][]string)

	for rows.Next() {
		var linkID int64
		var filter string

		err := rows.Scan(&linkID, &filter)
		if err != nil {
			return nil
		}

		if _, ok := filtersByID[linkID]; !ok {
			filtersByID[linkID] = []string{}
		}

		filtersByID[linkID] = append(filtersByID[linkID], filter)
	}

	return filtersByID
}

func (s *SQLStorage) GetLinks(ctx context.Context, id int64) ([]scrappertypes.LinkResponse, error) {
	rows, err := s.db.Query(ctx, `
        SELECT l.id, l.url, l.changed_at 
        FROM links l
        JOIN link_users lu ON l.id = lu.link_id
        JOIN users u ON u.id = lu.user_id
        WHERE u.telegram_id = $1`, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var links []scrappertypes.LinkResponse

	for rows.Next() {
		var link scrappertypes.LinkResponse

		err := rows.Scan(&link.ID, &link.URL, &link.LastChecked)
		if err != nil {
			return nil, err
		}

		links = append(links, link)
	}

	tags := s.GetTags(ctx, id)
	filters := s.GetFilters(ctx, id)

	for i := range links {
		links[i].Tags = tags[links[i].ID]
		links[i].Filters = filters[links[i].ID]
	}

	return links, nil
}

func (s *SQLStorage) IsURLInAdded(ctx context.Context, id int64, u string) bool {
	var exists bool

	err := s.db.QueryRow(ctx, `
        SELECT EXISTS (
            SELECT 1 FROM link_users lu
            JOIN users u ON u.id = lu.user_id
            JOIN links l ON l.id = lu.link_id
            WHERE u.telegram_id = $1 AND l.url = $2
        )`, id, u).Scan(&exists)
	if err != nil {
		slog.Error(ErrExecQuery.Error())
		slog.String("error", err.Error())

		return false
	}

	return exists
}

func (s *SQLStorage) GetBatchOfLinks(ctx context.Context, batch int, lastID int64) ([]scrappertypes.LinkResponse, int64) {
	var links []scrappertypes.LinkResponse

	rows, err := s.db.Query(ctx, `
			SELECT id, url FROM links
			WHERE id > $1
			ORDER BY id
			LIMIT $2`, lastID, batch)
	if err != nil {
		return []scrappertypes.LinkResponse{}, lastID
	}

	lastReturnedID := lastID

	for rows.Next() {
		var link scrappertypes.LinkResponse

		err := rows.Scan(&link.ID, &link.URL)
		if err != nil {
			return []scrappertypes.LinkResponse{}, lastID
		}

		links = append(links, link)
		lastReturnedID = link.ID
	}

	return links, lastReturnedID
}

func (s *SQLStorage) GetPreviousUpdate(ctx context.Context, ID int64) time.Time {
	var updTime time.Time

	err := s.db.QueryRow(ctx, `
		SELECT changed_at FROM links
		WHERE id = $1`, ID).Scan(&updTime)
	if err != nil {
		return time.Time{}
	}
	//TODO: отличать отсутствие строк от ошибки

	return updTime
}

func (s *SQLStorage) SaveLastUpdate(ctx context.Context, ID int64, updTime time.Time) error {
	_, err := s.db.Exec(ctx, `
        UPDATE links
        SET changed_at = $1
        WHERE id = $2
        `, updTime, ID)

	return err
}

func (s *SQLStorage) GetTgChatIDsForLink(ctx context.Context, link string) []int64 {
	rows, err := s.db.Query(ctx, `
        SELECT u.telegram_id
        FROM users u
        JOIN link_users lu ON u.id = lu.user_id
        JOIN links l ON l.id = lu.link_id
        WHERE l.url = $1`, link)
	if err != nil {
		return []int64{}
	}

	defer rows.Close()

	var tgIDs []int64

	for rows.Next() {
		var tgID int64

		err := rows.Scan(&tgID)
		if err != nil {
			return []int64{}
		}

		tgIDs = append(tgIDs, tgID)
	}

	return tgIDs
}

func (s *SQLStorage) Close() {
	s.db.Close()
}
