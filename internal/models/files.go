package models

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"slices"
	"strings"
	"time"

	"gofiles/utils"

	"github.com/google/uuid"
)

var (
	textFiles  = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv", "R", "r"}
	imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "svg", "ico", "heic", "raw"}
	videoFiles = []string{"mp4", "wav", "mp3", "aif", "aiff", "mov", "avi", "mkv", "flv", "wmv", "webm", "mpg", "mpeg", "3gp"}
)

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
	ID            int          `json:"id"`
	DirectoryID   uuid.UUID    `json:"directoryId"`
	Keywords      string       `json:"keywords"`
	SizeSimple    string       `json:"sizeSimple"`
	ModTimeSimple string       `json:"modTimeStr"`
	Type          string       `json:"type"`
	URL           template.URL `json:"url"`
	TSRank        float64      `json:"tsRank"`
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

	rows, err := conn.Query(stmt, name, limit, offset)
	if err != nil {
		return err
	}
	defer rows.Close()

	// go Explain(stmt, name, limit, offset)

	for rows.Next() {

		var rawJSON []byte
		var f FileData
		f.Keywords = name

		err := rows.Scan(
			&f.ID,
			&rawJSON,
			&f.TSRank,
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

	err = rows.Err()
	if err != nil {
		return err
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

	stmt := fmt.Sprintf(query, language)

	rows, err := conn.Query(stmt, qp.Keywords, qp.Limit, qp.Offset)
	if err != nil {
		return err
	}

	// go Explain(stmt, name, limit, offset)

	for rows.Next() {

		var d FileData
		d.Keywords = qp.Keywords

		rawJSON := []byte{}

		err := rows.Scan(
			&d.ID,
			&rawJSON,
			&d.TSRank,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawJSON, &d.FinfoJSON)

		d.SimplifyDetails()

		if err != nil {
			return err
		}

		flist.List = append(flist.List, d)
	}

	if err := rows.Err(); err != nil {
		return err
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
	order := "DESC"

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

	stmt := fmt.Sprintf(query, language, clause, order)
	println("AppendList stmt:", stmt)

	rows, err := conn.Query(stmt, qp.Keywords, qp.Limit, qp.Offset)
	if err != nil {
		return err
	}

	go Explain(stmt, qp.Keywords, qp.Limit, qp.Offset)

	for rows.Next() {

		tsRank := float64(0)
		var id int
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

		dataWithID := FileData{
			FinfoJSON: data,
			ID:        id,
		}

		dataWithID.Keywords = qp.Keywords
		dataWithID.SimplifyDetails()

		if err != nil {
			return err
		}

		flist.List = append(flist.List, dataWithID)
	}

	err = rows.Err()
	if err != nil {
		return err
	}

	return nil
}

func (f *FileData) SimplifyDetails() {
	f.SizeSimple = ConvertBytes(f.Size)
	f.ModTimeSimple = f.ModTime.Format("2006-01-02 15:04:05")
	f.CheckExtension()
}

func ConvertBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
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

	f.URL = template.URL(strings.ReplaceAll(url, "\\", "/"))
}

func (f *FileData) GetByID(id int) error {
	const stmt = `SELECT data, directory_id, keywords FROM files WHERE id=$1;`

	conn, err := utils.PgConn()
	if err != nil {
		return fmt.Errorf("error connecting to database; %w", err)
	}

	raw := []byte{}
	dirID := uuid.UUID{}
	keywords := ""

	err = conn.QueryRow(stmt, id).Scan(&raw, &dirID, &keywords)
	if err != nil {
		return fmt.Errorf("error querying database for file by ID; %w", err)
	}

	data := FinfoJSON{}

	err = json.Unmarshal(raw, &data)

	f.FinfoJSON = data
	f.ID = id
	f.DirectoryID = dirID
	f.Keywords = keywords

	// f.SimplifyDetails()

	if err != nil {
		return fmt.Errorf("error retrieving file by ID; %w", err)
	}

	return nil
}

func Explain(stmt string, placeholders ...any) {
	conn, err := utils.PgConn()
	if err != nil {
		log.Println("Error connecting to the database for search words:", err)
		return
	}

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
