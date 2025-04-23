package repository

import (
	"context"
	"errors"
	"fmt"
	"go-progira/internal/domain/types/scrappertypes"
	"log/slog"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v4"

	"github.com/jackc/pgx/v4/pgxpool"
)

type ORMLinkService struct {
	db *pgxpool.Pool
}

func (s *ORMLinkService) CreateChat(ctx context.Context, id int64) error {
	sql, args, err := buildInsertQuery("users",
		[]string{"telegram_id"},
		[]interface{}{id},
		"ON CONFLICT (telegram_id) DO NOTHING")

	if err != nil {
		return err
	}

	_, err = s.db.Exec(ctx, sql, args...)
	if err != nil {
		slog.Error(ErrCreateChat.Error(),
			slog.String("error", err.Error()))
	}

	return err
}

func (s *ORMLinkService) DeleteChat(_ context.Context, id int64) error {
	sql, args, err := sq.
		Delete("users").
		Where(sq.Eq{"telegram_id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build DELETE query",
			slog.String("error", err.Error()))

		return err
	}

	_, err = s.db.Exec(context.Background(), sql, args...)
	if err != nil {
		slog.Error(ErrDeleteChat.Error(),
			slog.String("error", err.Error()))
	}

	return err
}

func (s *ORMLinkService) getLinkID(ctx context.Context, url string) (int64, error) {
	tx, _ := s.db.Begin(ctx)

	sql, args, errBuildQuery := sq.
		Select("id").
		From("links").
		Where(sq.Eq{"url": url}).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	if errBuildQuery != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", errBuildQuery.Error()))

		return 0, errBuildQuery
	}

	var linkID int64

	errQuery := tx.QueryRow(ctx, sql, args...).Scan(&linkID)
	if errQuery != nil {
		slog.Error("Failed to retrieve link ID after conflict",
			slog.String("error", errQuery.Error()))

		_ = tx.Rollback(ctx)

		return 0, errQuery
	}

	return linkID, nil
}

func (s *ORMLinkService) AddLink(ctx context.Context, id int64, url string, tags, filters []string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	var linkID int64

	sql, args, err := buildInsertQuery("links",
		[]string{"url", "changed_at"},
		[]interface{}{url, sq.Expr("NOW()")},
		"ON CONFLICT DO NOTHING RETURNING id")

	if err != nil {
		return err
	}

	errQuery := tx.QueryRow(ctx, sql, args...).Scan(&linkID)

	if errors.Is(errQuery, pgx.ErrNoRows) {
		linkID, errQuery = s.getLinkID(ctx, url)
		if errQuery != nil {
			return errQuery
		}
	} else if errQuery != nil {
		slog.Error("Query Exec error",
			slog.String("error", errQuery.Error()))

		errRollback := tx.Rollback(ctx)
		if errRollback != nil {
			return errRollback
		}

		return errQuery
	}

	if linkID == 0 {
		slog.Error("Invalid ID",
			slog.Int("id", int(linkID)))
	}

	sql, args, err = buildInsertQuery("link_users",
		[]string{"user_id", "link_id"},
		[]interface{}{sq.Expr("(SELECT id FROM users WHERE telegram_id = ?)", id),
			linkID},
		"ON CONFLICT DO NOTHING")

	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, sql, args...)
	if err != nil {
		slog.Error(ErrExecQuery.Error(),
			slog.String("error", err.Error()))
	}

	err = s.addTags(ctx, linkID, tags)
	if err != nil {
		slog.Error("Error adding tags")
	}

	err = s.addFilters(ctx, linkID, filters)
	if err != nil {
		slog.Error("Error adding filters")
	}

	return tx.Commit(ctx)
}

func (s *ORMLinkService) RemoveLink(ctx context.Context, id int64, link string) error {
	userSubquery, userArgs, errUser := sq.Select("id").From("users").Where(sq.Eq{"telegram_id": id}).ToSql()
	linkSubquery, linkArgs, errLink := sq.Select("id").From("links").Where(sq.Eq{"url": link}).ToSql()

	allArgs := make([]interface{}, 0, len(userArgs)+len(linkArgs))
	allArgs = append(allArgs, userArgs...)
	allArgs = append(allArgs, linkArgs...)

	query, _, err := sq.Delete("link_users").
		Where(sq.Expr(fmt.Sprintf("user_id = (%s)", userSubquery))).
		Where(sq.Expr(fmt.Sprintf("link_id = (%s)", linkSubquery))).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	slog.Info("DELETE LINK",
		slog.String("query", query),
		slog.Any("args", allArgs))

	if errUser != nil || errLink != nil || err != nil {
		slog.Error("Unable to build DELETE query")

		return err
	}

	_, err = s.db.Exec(ctx, query, allArgs...)
	if err != nil {
		slog.Error(ErrRemoveLink.Error(),
			slog.String("error", err.Error()),
			slog.Int("id", int(id)))
	}

	return err
}

func getItems(rows pgx.Rows) (map[int64][]string, error) {
	itemsByID := make(map[int64][]string)

	for rows.Next() {
		var linkID int64

		var filter string

		err := rows.Scan(&linkID, &filter)
		if err != nil {
			slog.Error("Error scanning")
			return nil, err
		}

		if _, ok := itemsByID[linkID]; !ok {
			itemsByID[linkID] = []string{}
		}

		itemsByID[linkID] = append(itemsByID[linkID], filter)
	}

	return itemsByID, nil
}

func (s *ORMLinkService) GetTags(ctx context.Context, id int64) map[int64][]string {
	sql, args, err := sq.
		Select("lt.link_id", "t.name").
		From("tags t").
		Join("link_tags lt ON lt.tag_id = t.id").
		Join("link_users lu ON lt.link_id = lu.link_id").
		Join("users u ON u.id = lu.user_id").
		Where(sq.Eq{"u.telegram_id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return nil
	}

	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		slog.Error("Error executing query",
			slog.String("error", err.Error()))

		return nil
	}

	defer rows.Close()

	tagsByID, err := getItems(rows)
	if err != nil {
		return nil
	}

	return tagsByID
}

func (s *ORMLinkService) GetFilters(ctx context.Context, id int64) map[int64][]string {
	sql, args, err := sq.
		Select("lf.link_id", "f.name").
		From("filters f").
		Join("link_filters lf ON lf.filter_id = f.id").
		Join("link_users lu ON lf.link_id = lu.link_id").
		Join("users u ON u.id = lu.user_id").
		Where(sq.Eq{"u.telegram_id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return nil
	}

	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		slog.Error("Error executing query",
			slog.String("error", err.Error()))

		return nil
	}

	defer rows.Close()

	filtersByID, err := getItems(rows)
	if err != nil {
		return nil
	}

	return filtersByID
}

func (s *ORMLinkService) GetLinks(ctx context.Context, id int64) ([]scrappertypes.LinkResponse, error) {
	sql, args, err := sq.
		Select("l.id", "l.url", "l.changed_at").
		From("links l").
		Join("link_users lu ON l.id = lu.link_id").
		Join("users u ON u.id = lu.user_id").
		Where(sq.Eq{"u.telegram_id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return nil, err
	}

	slog.Info("SELECT query",
		slog.String("sql", sql),
		slog.Any("args", args))

	rows, err := s.db.Query(ctx, sql, args...)
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

func (s *ORMLinkService) IsURLInAdded(ctx context.Context, id int64, u string) bool {
	var exists bool

	subQuery := sq.
		Select("1").
		From("link_users lu").
		Join("users u ON u.id = lu.user_id").
		Join("links l ON l.id = lu.link_id").
		Where(sq.Eq{"u.telegram_id": id, "l.url": u}).
		PlaceholderFormat(sq.Dollar)

	sqlQuery, _, err := subQuery.ToSql()
	if err != nil {
		slog.Error("Unable to build subquery",
			slog.String("error", err.Error()))
		return false
	}

	mainQuery := sq.
		Select("EXISTS (" + sqlQuery + ")").
		PlaceholderFormat(sq.Dollar)

	query, args, err := mainQuery.ToSql()
	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return false
	}

	if err = s.db.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		slog.Error(ErrExecQuery.Error())
		slog.String("error", err.Error())

		return false
	}

	return exists
}

func (s *ORMLinkService) GetBatchOfLinks(ctx context.Context, batch int, lastID int64) (links []scrappertypes.LinkResponse,
	lastReturnedID int64) {
	if batch < 0 {
		slog.Error("batch cannot be negative",
			slog.Int("batch", batch))

		return []scrappertypes.LinkResponse{}, lastID
	}

	sql, args, err := sq.
		Select("id", "url").
		From("links").
		Where(sq.Gt{"id": lastID}).
		OrderBy("id").
		Limit(uint64(batch)).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	slog.Info("SELECT query",
		slog.String("sql", sql),
		slog.Any("args", args))

	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return []scrappertypes.LinkResponse{}, lastID
	}

	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		slog.Error("Query error",
			slog.String("error", err.Error()))

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

func (s *ORMLinkService) GetPreviousUpdate(ctx context.Context, id int64) time.Time {
	var updTime time.Time

	sql, args, err := sq.
		Select("changed_at").
		From("links").
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return time.Time{}
	}

	err = s.db.QueryRow(ctx, sql, args...).Scan(&updTime)
	if err != nil {
		slog.Error("Query error",
			slog.String("error", err.Error()))

		return time.Time{}
	}

	return updTime
}

func (s *ORMLinkService) SaveLastUpdate(ctx context.Context, id int64, updTime time.Time) error {
	sql, args, err := sq.Update("links").
		Set("changed_at", updTime).
		Where(sq.Eq{"id": id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build UPDATE query",
			slog.String("error", err.Error()))

		return err
	}

	_, err = s.db.Exec(ctx, sql, args...)
	if err != nil {
		slog.Error("Query error",
			slog.String("error", err.Error()))
	}

	return err
}

func (s *ORMLinkService) GetTgChatIDsForLink(ctx context.Context, link string) []int64 {
	sql, args, err := sq.
		Select("u.telegram_id").
		From("users u").
		Join("link_users lu ON u.id = lu.user_id").
		Join("links l ON l.id = lu.link_id").
		Where(sq.Eq{"l.url": link}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		slog.Error("Unable to build SELECT query",
			slog.String("error", err.Error()))

		return []int64{}
	}

	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		slog.Error("Query error",
			slog.String("error", err.Error()))

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

func (s *ORMLinkService) Close() {
	s.db.Close()
}

func (s *ORMLinkService) addTags(ctx context.Context, linkID int64, tags []string) error {
	tx, _ := s.db.Begin(ctx)

	for _, tag := range tags {
		var tagID int64

		sql, args, err := buildInsertQuery("tags",
			[]string{"name"},
			[]interface{}{tag},
			"ON CONFLICT (name) DO NOTHING RETURNING id")

		if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, sql, args...).Scan(&tagID)
		if errors.Is(err, pgx.ErrNoRows) {
			sql, args, errBuildQuery := sq.
				Select("id").
				From("tags").
				Where(sq.Eq{"name": tag}).
				PlaceholderFormat(sq.Dollar).
				ToSql()

			if errBuildQuery != nil {
				slog.Error("Unable to build SELECT query",
					slog.String("error", errBuildQuery.Error()))

				return errBuildQuery
			}

			errQuery := tx.QueryRow(ctx, sql, args...).Scan(&tagID)
			if errQuery != nil {
				slog.Error("Unable to execute query",
					slog.String("error", errQuery.Error()))

				return errQuery
			}
		} else if err != nil {
			return err
		}

		sql, args, err = buildInsertQuery("link_tags",
			[]string{"link_id", "tag_id"},
			[]interface{}{linkID, tagID},
			"ON CONFLICT DO NOTHING")

		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, sql, args...)

		if err != nil {
			slog.Error(ErrExecQuery.Error())
			slog.String("error", err.Error())

			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *ORMLinkService) addFilters(ctx context.Context, linkID int64, filters []string) error {
	tx, _ := s.db.Begin(ctx)

	for _, filter := range filters {
		var filterID int64

		sql, args, err := buildInsertQuery("filters",
			[]string{"name"},
			[]interface{}{filter},
			"ON CONFLICT (name) DO NOTHING RETURNING id")

		if err != nil {
			return err
		}

		err = tx.QueryRow(ctx, sql, args...).Scan(&filterID)
		if errors.Is(err, pgx.ErrNoRows) {
			sql, args, errBuildQuery := sq.
				Select("id").
				From("filters").
				Where(sq.Eq{"name": filter}).
				PlaceholderFormat(sq.Dollar).
				ToSql()

			if errBuildQuery != nil {
				slog.Error("Unable to build SELECT query",
					slog.String("error", errBuildQuery.Error()))

				return errBuildQuery
			}

			errQuery := tx.QueryRow(ctx, sql, args...).Scan(&filterID)
			if errQuery != nil {
				slog.Error("Unable to execute query",
					slog.String("error", errQuery.Error()))

				return errQuery
			}
		} else if err != nil {
			return err
		}

		sql, args, err = buildInsertQuery("link_filters",
			[]string{"link_id", "filter_id"},
			[]interface{}{linkID, filterID},
			"ON CONFLICT DO NOTHING")

		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx, sql, args...)

		if err != nil {
			slog.Error(ErrExecQuery.Error(),
				slog.String("error", err.Error()))
		}
	}

	return tx.Commit(ctx)
}

func buildInsertQuery(table string, columns []string, values []interface{}, suffix string) (sql string,
	args []interface{}, err error) {
	sql, args, err = sq.
		Insert(table).
		Columns(columns...).
		Values(values...).
		Suffix(suffix).
		PlaceholderFormat(sq.Dollar).
		ToSql()

	slog.Info("INSERT query",
		slog.String("sql", sql),
		slog.Any("args", args))

	if err != nil {
		slog.Error("Unable to build INSERT query",
			slog.String("error", err.Error()))
	}

	return sql, args, err
}
