package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileModel struct {
	DB  *pgxpool.Pool
	Log *slog.Logger
}

type Filer interface {
	Row(id int) (f FileData, err error)
	Rows(keywords string, limit int, offset int) (list []FileData, count int, err error)
	AppendListParams(qp QueryParams) (list []FileData, err error)
}

func (m *FileModel) Explain(stmt string, args pgx.NamedArgs) {
	if m == nil || m.Log == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := m.DB.Query(ctx, fmt.Sprintf("EXPLAIN ANALYZE %s", stmt), args)
	if err != nil {
		m.Log.Error("running EXPLAIN ANALYZE:", "error", err)
		return
	}

	// buf := strings.Builder{}
	// defer rows.Close()
	// for rows.Next() {
	// 	var line string
	// 	if err := rows.Scan(&line); err != nil {
	// 		_, _ = buf.WriteString(err.Error())
	// 	}
	// 	_, _ = buf.WriteString("   ")
	// 	_, _ = buf.WriteString(strings.TrimSpace(line))
	// }

	// if err := rows.Err(); err != nil {
	// 	_, _ = buf.WriteString(err.Error())
	// }

	lastTwo := make([]string, 0)
	defer rows.Close()
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			lastTwo = append(lastTwo, err.Error())
		}
		if strings.Contains(line, "Planning") || strings.Contains(line, "Execution") {
			lastTwo = append(lastTwo, strings.TrimSpace(line))
		}
	}

	if err := rows.Err(); err != nil {
		lastTwo = append(lastTwo, err.Error())
	}

	msg := strings.Join(lastTwo, "; ")

	m.Log.Debug("explain", "analyze", msg)
}

func (m *FileModel) Row(ID int) (f FileData, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	stmt := `SELECT data, directory_id, keywords FROM files WHERE id = @ID;`

	args := pgx.NamedArgs{
		"ID": ID,
	}

	raw := []byte{}
	dirID := uuid.UUID{}
	keywords := ""

	err = m.DB.QueryRow(ctx, stmt, args).Scan(&raw, &dirID, &keywords)
	if err != nil {
		return f, fmt.Errorf("error querying database for file by ID; %w", err)
	}

	data := FinfoJSON{}

	err = json.Unmarshal(raw, &data)

	f.FinfoJSON = data
	f.ID = ID
	f.DirectoryID = dirID
	f.Keywords = keywords

	f.SimplifyDetails()

	if err != nil {
		return f, fmt.Errorf("error retrieving file by ID; %w", err)
	}

	return f, nil
}

func (m *FileModel) Rows(keywords string, limit int, offset int) (list []FileData, count int, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	m.Log.Debug("searching files", "keywords", keywords)

	stmt := `SELECT DISTINCT(id), directory_id, COUNT(*) OVER () AS count, data, ts_rank_cd( to_tsvector('polish', @keywords), 
	websearch_to_tsquery('polish', @keywords) ) as ts_rank FROM files 
	WHERE websearch_to_tsquery('polish', @keywords) @@ to_tsvector('polish', keywords) 
	ORDER BY ts_rank DESC, id asc LIMIT @limit OFFSET @offset;`

	args := pgx.NamedArgs{
		"keywords": keywords,
		"limit":    limit,
		"offset":   offset,
	}

	rows, err := m.DB.Query(ctx, stmt, args)
	if err != nil {
		if m.Log != nil {
			m.Log.Error("files", "rows", err)
		}
		return list, 0, err
	}
	defer rows.Close()

	go m.Explain(stmt, args)

	for rows.Next() {

		var rawJSON []byte
		var f FileData
		f.Keywords = keywords

		err := rows.Scan(
			&f.ID,
			&f.DirectoryID,
			&count,
			&rawJSON,
			&f.TSRank,
		)
		if err != nil {
			// TO DO sentinel ERR
			return list, 0, err
		}

		err = json.Unmarshal(rawJSON, &f.FinfoJSON)
		if err != nil {
			// TO DO sentinel ERR
			return list, 0, err
		}

		f.SimplifyDetails()

		list = append(list, f)
	}

	err = rows.Err()
	if err != nil {
		// TO DO sentinel ERR
		return list, count, err
	}

	return list, count, nil
}

func (m *FileModel) AppendListParams(qp QueryParams) (list []FileData, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var query string
	var clause string
	order := "DESC"

	switch qp.Order {
	case "name":
		clause = "data->>'name'"
	case "size":
		clause = "(data->>'size')::bigint"
	case "modtime":
		clause = "(data->>'modTime')::timestamp"
	default:
		clause = ""
	}
	if qp.Ascending {
		order = "ASC"
	}

	withClause := len(clause) > 0

	query = `SELECT DISTINCT(id), directory_id, data, ts_rank_cd( to_tsvector('polish', keywords),
	websearch_to_tsquery('polish', @Keywords) ) as ts_rank FROM files
	WHERE websearch_to_tsquery('polish', @Keywords) @@ to_tsvector('polish', keywords)
	ORDER BY ts_rank DESC, id ASC LIMIT @Limit OFFSET @Offset;`
	// default
	stmt := query

	if withClause {
		query = `
		SELECT 
		DISTINCT(id), directory_id, data, ts_rank_cd( to_tsvector('polish', keywords), websearch_to_tsquery('polish', @Keywords) ) as ts_rank, %[1]s
		FROM files
		WHERE websearch_to_tsquery('polish', @Keywords) @@ to_tsvector('polish', keywords)
		ORDER BY %[1]s %[2]s, ts_rank DESC, id ASC
		LIMIT @Limit OFFSET @Offset;`
		// with parameters
		stmt = fmt.Sprintf(query, clause, order)
	}

	m.Log.Debug("append list", "query", stmt)

	args := pgx.NamedArgs{
		"Keywords": qp.Keywords,
		"Limit":    qp.Limit,
		"Offset":   qp.Offset,
	}

	rows, err := m.DB.Query(ctx, stmt, args)
	if err != nil {
		return list, err
	}

	go m.Explain(stmt, args)

	for rows.Next() {

		var rawJSON []byte
		var f FileData
		f.Keywords = qp.Keywords

		var (
			errRows error
			_drop   any
		)

		switch qp.Order {
		case "name", "size", "modtime":
			errRows = rows.Scan(
				&f.ID,
				&f.DirectoryID,
				&rawJSON,
				&f.TSRank,
				&_drop,
			)
		default:
			errRows = rows.Scan(
				&f.ID,
				&f.DirectoryID,
				&rawJSON,
				&f.TSRank,
			)
		}

		if errRows != nil {
			m.Log.Debug("append", "query", errRows)
			return list, err
		}

		err = json.Unmarshal(rawJSON, &f.FinfoJSON)
		if err != nil {
			return list, err
		}

		f.SimplifyDetails()

		list = append(list, f)
	}

	err = rows.Err()
	if err != nil {
		// TO DO sentinel ERR
		return list, err
	}

	return list, nil
}
