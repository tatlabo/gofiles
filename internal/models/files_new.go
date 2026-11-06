package models

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileModel struct {
	DB *pgxpool.Pool
}

func (m *FileModel) GetList(keywords string, limit int, offset int) (list []FileData, count int, err error) {
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

// func (flist *FileModel) AppendList(qp QueryParams) error {
// 	const language = "'polish'"

// 	query := `
// 	SELECT DISTINCT(id), data, ts_rank_cd( to_tsvector(%[1]s, keywords),
// 	websearch_to_tsquery(%[1]s, $1) ) as ts_rank FROM files
// 	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
// 	ORDER BY ts_rank DESC, id ASC LIMIT $2 OFFSET $3;`

// 	conn, err := utils.PgConn()
// 	if err != nil {
// 		return err
// 	}

// 	stmt := fmt.Sprintf(query, language)

// 	rows, err := conn.Query(stmt, qp.Keywords, qp.Limit, qp.Offset)
// 	if err != nil {
// 		return err
// 	}

// 	// go Explain(stmt, name, limit, offset)

// 	for rows.Next() {

// 		var d FileData
// 		d.Keywords = qp.Keywords

// 		rawJSON := []byte{}

// 		err := rows.Scan(
// 			&d.ID,
// 			&rawJSON,
// 			&d.TSRank,
// 		)
// 		if err != nil {
// 			return err
// 		}

// 		err = json.Unmarshal(rawJSON, &d.FinfoJSON)

// 		d.SimplifyDetails()

// 		if err != nil {
// 			return err
// 		}

// 		flist.List = append(flist.List, d)
// 	}

// 	if err := rows.Err(); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (flist *FileModel) AppendListParams(qp QueryParams) error {
// 	const language = "'polish'"

// 	var query string
// 	var clause string
// 	order := "DESC"

// 	switch qp.Order {
// 	case "name":
// 		clause = "data->>'name'"
// 	case "size":
// 		clause = "(data->>'size')::bigint"
// 	case "modtime":
// 		clause = "(data->>'modTime')::timestamp"
// 	}
// 	if qp.Ascending {
// 		order = "ASC"
// 	}

// 	query = `
// 	SELECT
// 	DISTINCT(id), data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank, %[2]s
// 	FROM files
// 	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
// 	ORDER BY %[2]s %[3]s, ts_rank DESC, id ASC
// 	LIMIT $2 OFFSET $3;`

// 	conn, err := utils.PgConn()
// 	if err != nil {
// 		return err
// 	}

// 	stmt := fmt.Sprintf(query, language, clause, order)
// 	println("AppendList stmt:", stmt)

// 	rows, err := conn.Query(stmt, qp.Keywords, qp.Limit, qp.Offset)
// 	if err != nil {
// 		return err
// 	}

// 	go Explain(stmt, qp.Keywords, qp.Limit, qp.Offset)

// 	for rows.Next() {

// 		tsRank := float64(0)
// 		var id int
// 		data := FinfoJSON{}

// 		rawData := []byte{}

// 		_drop := ""

// 		err := rows.Scan(
// 			&id,
// 			&rawData,
// 			&tsRank,
// 			&_drop,
// 		)
// 		if err != nil {
// 			return err
// 		}

// 		err = json.Unmarshal(rawData, &data)

// 		dataWithID := FileData{
// 			FinfoJSON: data,
// 			ID:        id,
// 		}

// 		dataWithID.Keywords = qp.Keywords
// 		dataWithID.SimplifyDetails()

// 		if err != nil {
// 			return err
// 		}

// 		flist.List = append(flist.List, dataWithID)
// 	}

// 	err = rows.Err()
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (flist *FileModel) SelectCount(name string) error {
// 	const language = "'polish'"
// 	stmt := fmt.Sprintf(`
// 	SELECT COUNT(DISTINCT id) FROM files
// 	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, language)

// 	conn, err := utils.PgConn()
// 	if err != nil {
// 		return fmt.Errorf("no connection to database")
// 	}

// 	err = conn.QueryRow(stmt, name).Scan(&flist.Count)
// 	if err != nil {
// 		return fmt.Errorf("where is error: there is no connection to database. %w", err)
// 	}

// 	return nil
// }
