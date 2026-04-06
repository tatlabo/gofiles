package models

import (
	"encoding/json"
	"fmt"
	"gofiles/utils"
	"html/template"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

var textFiles = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv", "R", "r"}
var imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "svg", "ico", "heic", "raw"}
var videoFiles = []string{"mp4", "wav", "mp3", "aif", "aiff", "mov", "avi", "mkv", "flv", "wmv", "webm", "mpg", "mpeg", "3gp"}

type FinfoJSON struct {
	Directory string    `json:"directory"`
	Name      string    `json:"name"`
	Ext       string    `json:"ext"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modTime"`
}

type FileData struct {
	FinfoJSON     `json:"finfo"`
	Id            uuid.UUID    `json:"id"`
	DirectoryId   uuid.UUID    `json:"directoryId"`
	Keywords      string       `json:"keywords"`
	SizeSimple    string       `json:"sizeSimple"`
	ModTimeSimple string       `json:"modTimeStr"`
	Type          string       `json:"type"`
	Url           template.URL `json:"url"`
	TsRank        float64      `json:"tsRank"`
}

type FilesDataList struct {
	List  []FileData
	Count int
}

func (flist *FilesDataList) GetList(name string, limit int, offset int) error {

	const language = "'polish'"
	const query = `
	SELECT 
	DISTINCT(id), data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
	FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY ts_rank DESC, id ASC
	LIMIT $2 OFFSET $3;`

	stmt := fmt.Sprintf(query, language)

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(stmt, name, limit, offset)
	if err != nil {
		return err
	}

	// go Explain(stmt, name, limit, offset)

	for rows.Next() {

		var rawJSON []byte
		var f FileData
		f.Keywords = name

		err := rows.Scan(
			&f.Id,
			&rawJSON,
			&f.TsRank,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawJSON, &f.FinfoJSON)
		if err != nil {
			return err
		}

		flist.List = append(flist.List, f)
	}

	return nil
}

func (flist *FilesDataList) AppendList(qp QueryParams) error {

	const language = "'polish'"

	query := `
	SELECT DISTINCT(id), data, ts_rank_cd( to_tsvector(%[1]s, keywords),
	websearch_to_tsquery(%[1]s, $1) ) as ts_rank FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY ts_rank DESC, id ASC LIMIT $2 OFFSET $3;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	stmt := fmt.Sprintf(query, language)

	rows, err := conn.Query(stmt, qp.Keywords, qp.Limit, qp.Offset)
	if err != nil {
		return err
	}

	// go Explain(stmt, name, limit, offset)

	for rows.Next() {

		var d FileData
		d.Keywords = qp.Keywords

		rawJson := []byte{}

		err := rows.Scan(
			&d.Id,
			&rawJson,
			&d.TsRank,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawJson, &d.FinfoJSON)

		d.SimplifyDetails()

		if err != nil {
			return err
		}

		flist.List = append(flist.List, d)
	}

	return nil
}

type QueryParams struct {
	Keywords  string
	Limit     int
	Offset    int
	Order     string
	Ascending bool
	Method    string
}

func (flist *FilesDataList) AppendListParams(qp QueryParams) error {

	const language = "'polish'"

	var query string
	var clause string
	var order = "DESC"

	switch qp.Order {
	case "name":
		clause = "data->>'name'"
	case "size":
		clause = "(data->>'size')::bigint"
	case "modtime":
		clause = "(data->>'modTime')::timestamp"
	}
	if qp.Ascending {
		order = "ASC"
	}

	query = `
	SELECT 
	DISTINCT(id), data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank, %[2]s
	FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY %[2]s %[3]s, ts_rank DESC, id ASC
	LIMIT $2 OFFSET $3;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	stmt := fmt.Sprintf(query, language, clause, order)

	rows, err := conn.Query(stmt, qp.Keywords, qp.Limit, qp.Offset)
	if err != nil {
		return err
	}

	// go Explain(stmt, name, limit, offset)

	for rows.Next() {

		tsRank := float64(0)
		id := uuid.UUID{}
		data := FinfoJSON{}

		rawData := []byte{}

		_drop := ""

		err := rows.Scan(
			&id,
			&rawData,
			&tsRank,
			&_drop,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawData, &data)

		dataWithId := FileData{
			FinfoJSON: data,
			Id:        id,
		}

		dataWithId.Keywords = qp.Keywords
		dataWithId.SimplifyDetails()

		if err != nil {
			return err
		}

		flist.List = append(flist.List, dataWithId)
	}

	return nil
}

func (flist *FilesDataList) SelectCount(name string) error {

	const language = "'polish'"
	var stmt = fmt.Sprintf(`
	SELECT COUNT(DISTINCT id) FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, language)

	conn, err := utils.PgConn()
	if err != nil {
		return fmt.Errorf("There is no connection to database")
	}
	defer conn.Close()

	err = conn.QueryRow(stmt, name).Scan(&flist.Count)

	if err != nil {
		errMsg := fmt.Errorf("There is error: there is no connection to database.\n%v", err.Error())
		return errMsg
	}

	return nil
}

func (f *FileData) SimplifyDetails() {
	f.SizeSimple = utils.ConvertBytes(f.Size)
	f.ModTimeSimple = f.ModTime.Format("2006-01-02 15:04:05")
	f.CheckExtension()
}

func (f *FileData) CheckExtension() {

	var url string
	f.Type = ""

	if slices.Contains(textFiles, strings.ToLower(f.Ext)) {
		f.Type = "txt"
		url = fmt.Sprintf("%s/%s.%v", f.Directory, f.Name, f.Ext)
	} else if slices.Contains(imageFiles, strings.ToLower(f.Ext)) {
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
		f.Type = "image"
	} else if slices.Contains(videoFiles, strings.ToLower(f.Ext)) {
		f.Type = "video"
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
	} else {
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
	}

	f.Url = template.URL(strings.ReplaceAll(url, "\\", "/"))

}

func (f *FileData) GetById(id uuid.UUID) error {

	const stmt = `SELECT data, directory_id, keywords FROM files WHERE id=$1;`

	conn, err := utils.PgConn()
	if err != nil {
		return fmt.Errorf("Error connecting to database:\n%v", err)
	}
	defer conn.Close()

	raw := []byte{}
	dirId := uuid.UUID{}
	keywords := ""

	err = conn.QueryRow(stmt, id).Scan(&raw, &dirId, &keywords)
	if err != nil {
		return fmt.Errorf("Error querying database for file by ID:\n%v", err)
	}

	data := FinfoJSON{}

	err = json.Unmarshal(raw, &data)

	f.FinfoJSON = data
	f.Id = id
	f.DirectoryId = dirId
	f.Keywords = keywords

	// f.SimplifyDetails()

	if err != nil {
		return fmt.Errorf("Error retrieving file by ID:\n%v", err)
	}

	return nil
}

// Constructor functions to ensure maps are initialized

// NewIndexedDirs creates a new IndexedDirs with initialized maps
func NewIndexedDirs() *IndexedDirs {
	return &IndexedDirs{
		Indexeddirs: make([]IndexedDir, 0),
		Params:      make(map[string]string),
		Error:       make(map[string]string),
	}
}

// Delete removes an indexed directory by ID
func (i *IndexedDirs) Delete(id string) error {

	query := `DELETE FROM directory WHERE id = $1;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	result, err := conn.Exec(fmt.Sprintf(`%s, %s;`, query, id))
	if err != nil {
		return fmt.Errorf("failed to delete indexed directory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no directory found with ID %v", id)
	}

	return nil
}

// DeleteByPath removes an indexed directory by path (alternative method)
func (i *IndexedDirs) DeleteByPath(path string) error {
	query := `DELETE FROM indexed WHERE path = $1;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	result, err := conn.Exec(query, path)
	if err != nil {
		return fmt.Errorf("failed to delete indexed directory: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no directory found with path %s", path)
	}

	return nil
}

func Explain(stmt string, placeholders ...any) {
	conn, err := utils.PgConn()
	if err != nil {
		log.Println("Error connecting to the database for search words:", err)
		return
	}
	defer conn.Close()
	explainAnalyze := fmt.Sprintf(`EXPLAIN ANALYZE %s`, stmt)
	fmt.Println()
	fmt.Printf("%v %v\n", explainAnalyze, placeholders)
	explainRows, err := conn.Query(explainAnalyze, placeholders...)
	if err != nil {
		log.Println("Error running EXPLAIN ANALYZE:", err)
	}
	defer explainRows.Close()
	fmt.Println()
	for explainRows.Next() {
		var line string
		if err := explainRows.Scan(&line); err != nil {
			log.Println("Error scanning EXPLAIN ANALYZE line:", err)
			continue
		}
		fmt.Println(line)
	}
	fmt.Println()
	if err := explainRows.Err(); err != nil {
		fmt.Println("Error iterating EXPLAIN ANALYZE rows:", err)
	}
}
