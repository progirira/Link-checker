package repository

import (
	"context"
	"fmt"
	"go-progira/internal/domain/types/scrappertypes"
	repository "go-progira/internal/repository/dictionary_storage"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

func NewLinkService(typeOfService, dbURL string) (repository.LinkService, error) {
	var pool *pgxpool.Pool

	var err error

	pool, err = pgxpool.Connect(context.Background(), dbURL)
	if err != nil {
		slog.Error(ErrPoolCreate.Error(),
			slog.String("error", err.Error()),
			slog.String("database URL", dbURL))

		return nil, err
	}

	switch typeOfService {
	case "orm":
		return &ORMLinkService{db: pool}, nil
	case "sql":
		return &SQLLinkService{db: pool}, nil
	default:
		return nil, fmt.Errorf("no such sql storage type: %s", typeOfService)
	}
}

type SQLLinkService struct {
	db *pgxpool.Pool
}

func (s *SQLLinkService) CreateChat(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, "INSERT INTO users (telegram_id) VALUES ($1) ON CONFLICT (telegram_id) DO NOTHING", id)
	if err != nil {
		slog.Error(ErrCreateChat.Error(),
			slog.String("error", err.Error()))
	}

	return err
}

func (s *SQLLinkService) DeleteChat(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, "DELETE FROM users WHERE telegram_id = $1", id)
	if err != nil {
		slog.Error(ErrDeleteChat.Error(),
			slog.String("error", err.Error()))
	}

	return err
}

func (s *SQLLinkService) AddLink(ctx context.Context, id int64, url string, tags, filters []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	var linkID int64

	_, errQuery := tx.Exec(ctx,
		"INSERT INTO links (url, changed_at) VALUES ($1, NOW()) ON CONFLICT (url) DO NOTHING", url)
	if errQuery != nil {
		slog.Error("Query Exec error" + errQuery.Error())

		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return errRollback
		}

		return errQuery
	}

	linkID, err = s.GetLinkIDByURL(ctx, url)

	if err != nil {
		slog.Error("Query Exec error" + err.Error())

		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return errRollback
		}

		return err
	}

	_, err = tx.Exec(ctx,
		"INSERT INTO link_users (user_id, link_id) VALUES ((SELECT id FROM users WHERE telegram_id = $1), $2) ON CONFLICT DO NOTHING",
		id, linkID)
	if err != nil {
		slog.Error(ErrExecQuery.Error() + err.Error())
	}

	var elementID int64

	for _, tag := range tags {
		err := tx.QueryRow(ctx, "INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id", tag).Scan(&elementID)
		if err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO link_tags (link_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", linkID, elementID)

			if err != nil {
				slog.Error(ErrExecQuery.Error() + err.Error())
			}
		}
	}

	for _, filter := range filters {
		err := tx.QueryRow(ctx, "INSERT INTO filters (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id", filter).Scan(&elementID)
		if err == nil {
			_, err = tx.Exec(ctx, "INSERT INTO link_filters (link_id, filter_id) VALUES ($1, $2) ON CONFLICT DO NOTHING", linkID, elementID)

			if err != nil {
				slog.Error(ErrExecQuery.Error() + err.Error())
			}
		}
	}

	return tx.Commit(ctx)
}

func (s *SQLLinkService) GetLinkIDByURL(ctx context.Context, url string) (linkID int64, err error) {
	err = s.db.QueryRow(ctx, "SELECT id FROM links WHERE url = $1", url).Scan(&linkID)
	if err != nil {
		slog.Error(ErrExecQuery.Error() + err.Error())
	}

	return linkID, err
}

func (s *SQLLinkService) RemoveLink(ctx context.Context, id int64, link string) error {
	_, err := s.db.Exec(ctx, `
        DELETE FROM link_users 
        WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1) 
        AND link_id = (SELECT id FROM links WHERE url = $2)`, id, link)
	if err != nil {
		slog.Error(ErrRemoveLink.Error(),
			slog.String("error", err.Error()),
			slog.Int("id", int(id)))
	}

	return err
}

func (s *SQLLinkService) GetTags(ctx context.Context, id int64) map[int64][]string {
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

func (s *SQLLinkService) GetFilters(ctx context.Context, id int64) map[int64][]string {
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

func (s *SQLLinkService) GetLinks(ctx context.Context, id int64) ([]scrappertypes.LinkResponse, error) {
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

func (s *SQLLinkService) IsURLInAdded(ctx context.Context, id int64, u string) bool {
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

func (s *SQLLinkService) GetBatchOfLinks(ctx context.Context, batch int, lastID int64) (links []scrappertypes.LinkResponse,
	lastReturnedID int64) {
	rows, err := s.db.Query(ctx, `
			SELECT id, url FROM links
			WHERE id > $1
			ORDER BY id
			LIMIT $2`, lastID, batch)
	if err != nil {
		return []scrappertypes.LinkResponse{}, lastID
	}

	lastReturnedID = lastID

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

func (s *SQLLinkService) GetPreviousUpdate(ctx context.Context, id int64) time.Time {
	var updTime time.Time

	err := s.db.QueryRow(ctx, `
		SELECT changed_at FROM links
		WHERE id = $1`, id).Scan(&updTime)
	if err != nil {
		return time.Time{}
	}

	return updTime
}

func (s *SQLLinkService) SaveLastUpdate(ctx context.Context, id int64, updTime time.Time) error {
	_, err := s.db.Exec(ctx, `
        UPDATE links
        SET changed_at = $1
        WHERE id = $2
        `, updTime, id)

	return err
}

func (s *SQLLinkService) GetTgChatIDsForLink(ctx context.Context, link string) []int64 {
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

func (s *SQLLinkService) Close() {
	s.db.Close()
}
