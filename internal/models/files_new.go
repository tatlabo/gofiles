package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileModel struct {
	DB *pgxpool.Pool
}

type Filer interface {
	Row(id int) (f FileData, err error)
	Rows(keywords string, limit int, offset int) (list []FileData, count int, err error)
}

func (m *FileModel) Row(id int) (f FileData, err error) {
	ctx := context.Context(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	stmt := `SELECT data, directory_id, keywords FROM files WHERE id=$1;`

	raw := []byte{}
	dirID := uuid.UUID{}
	keywords := ""

	err = m.DB.QueryRow(ctx, stmt, id).Scan(&raw, &dirID, &keywords)
	if err != nil {
		return f, fmt.Errorf("error querying database for file by ID; %w", err)
	}

	data := FinfoJSON{}

	err = json.Unmarshal(raw, &data)

	f.FinfoJSON = data
	f.ID = id
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
		slog.Error(err.Error())
		return list, 0, err
	}
	defer rows.Close()

	// go Explain(stmt, name, limit, offset)

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

		list = append(list, f)
	}

	err = rows.Err()
	if err != nil {
		// TO DO sentinel ERR
		return list, count, err
	}

	return list, count, nil
}
