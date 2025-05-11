package repository

import (
	"context"
	"errors"
	"fmt"
	"go-progira/internal/domain/types/scrappertypes"
	repository "go-progira/internal/repository/dictionary_storage"
	"go-progira/pkg/e"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v4"

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

	errScan := tx.QueryRow(ctx, `SELECT id FROM links WHERE url = $1`, url).Scan(&linkID)
	if !errors.Is(errScan, pgx.ErrNoRows) {
		return e.ErrLinkAlreadyExists
	}

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

	linkID, err = s.getLinkIDByURL(ctx, tx, url)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO link_users (user_id, link_id) 
				VALUES ((SELECT id FROM users WHERE telegram_id = $1), $2)
				ON CONFLICT DO NOTHING`,
		id, linkID)
	if err != nil {
		slog.Error(ErrExecQuery.Error() + err.Error())
	}

	err = s.saveTags(ctx, tx, id, linkID, tags)
	if err != nil {
		return err
	}

	err = s.saveFilters(ctx, tx, id, linkID, filters)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (s *SQLLinkService) getLinkIDByURL(ctx context.Context, tx pgx.Tx, url string) (linkID int64, err error) {
	err = tx.QueryRow(ctx, "SELECT id FROM links WHERE url = $1", url).Scan(&linkID)
	if err != nil {
		slog.Error("Query Exec error" + err.Error())

		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return 0, errRollback
		}

		return 0, err
	}

	return linkID, err
}

func (s *SQLLinkService) saveTags(ctx context.Context, tx pgx.Tx, userID, linkID int64, tags []string) error {
	var elementID int64

	var err error

	for _, tag := range tags {
		_, errInsert := tx.Exec(ctx, "INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", tag)
		errSelect := tx.QueryRow(ctx, "SELECT id FROM tags WHERE name = $1", tag).Scan(&elementID)

		if errInsert == nil && errSelect == nil {
			_, err = tx.Exec(ctx, `INSERT INTO link_tags (link_id, tag_id, user_id) 
						VALUES ($1, $2, (SELECT id FROM users WHERE telegram_id = $3)) 
						ON CONFLICT DO NOTHING`, linkID, elementID, userID)
			if err != nil {
				slog.Error(ErrExecQuery.Error() + err.Error())
			}
		} else {
			slog.Error(ErrExecQuery.Error())
		}
	}

	return nil
}

func (s *SQLLinkService) saveFilters(ctx context.Context, tx pgx.Tx, userID, linkID int64, filters []string) error {
	var elementID int64

	var err error

	for _, filter := range filters {
		_, errInsert := tx.Exec(ctx, "INSERT INTO filters (name) VALUES ($1) ON CONFLICT (name) DO NOTHING", filter)
		errSelect := tx.QueryRow(ctx, "SELECT id FROM filters WHERE name = $1", filter).Scan(&elementID)

		if errInsert == nil && errSelect == nil {
			_, err = tx.Exec(ctx, `INSERT INTO link_filters (link_id, filter_id, user_id)
					VALUES ($1, $2, (SELECT id FROM users WHERE telegram_id = $3))
					ON CONFLICT DO NOTHING`, linkID, elementID, userID)

			if err != nil {
				slog.Error(ErrExecQuery.Error() + err.Error())
			}
		} else {
			slog.Error(ErrExecQuery.Error())
		}
	}

	return nil
}

func (s *SQLLinkService) RemoveLink(ctx context.Context, id int64, link string) error {
	var linkID int64

	err := s.db.QueryRow(ctx, `SELECT id FROM links WHERE url = $1`, link).Scan(&linkID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return e.ErrLinkNotFound
		}

		slog.Error("Error while getting link ID: " + err.Error())

		return e.ErrDeleteLink
	}

	res, err := s.db.Exec(ctx, `
        DELETE FROM link_users 
        WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1) 
        AND link_id = $2`, id, linkID)
	if err != nil {
		slog.String("Error while deleting link: ", err.Error())

		return e.ErrDeleteLink
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		return e.ErrLinkNotFound
	}

	_, err = s.db.Exec(ctx, `
	DELETE FROM links
	WHERE id = $1 AND NOT EXISTS (SELECT 1 FROM link_users WHERE link_id = $1)`, linkID)
	if err != nil {
		slog.String("Error while deleting link from link_users: ", err.Error())

		return e.ErrDeleteLink
	}

	return nil
}

func (s *SQLLinkService) GetTags(ctx context.Context, id int64) map[int64][]string {
	rows, err := s.db.Query(ctx, `
        SELECT lt.link_id, t.name 
        FROM tags t
        JOIN link_tags lt ON lt.tag_id = t.id
        JOIN users u ON u.id = lt.user_id
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
        JOIN users u ON u.id = lf.user_id
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

func (s *SQLLinkService) DeleteTag(ctx context.Context, id int64, tag string) error {
	res, err := s.db.Exec(ctx, `
        DELETE FROM link_tags
        WHERE user_id = (SELECT id FROM users WHERE telegram_id = $1) 
        AND tag_id = (SELECT id FROM tags WHERE name = $2)`, id, tag)

	if err != nil {
		slog.Error("Error deleting tag: " + err.Error())

		return err
	}

	rowsAffected := res.RowsAffected()
	if rowsAffected == 0 {
		return e.ErrTagNotFound
	}

	return nil
}

func (s *SQLLinkService) Close() {
	s.db.Close()
}
