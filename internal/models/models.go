package models

import (
	"encoding/json"
	"fmt"
	"gofiles/utils"
	"html/template"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

var textFiles = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv", "R", "r"}
var imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "svg", "ico", "heic", "raw"}
var videoFiles = []string{"mp4", "wav", "mp3", "aif", "aiff"}

type Finfo struct {
	Id          uuid.UUID    `db:"id"`
	DirectoryId uuid.UUID    `json:"directoryId"`
	Directory   string       `json:"directory"`
	Name        string       `json:"name"`
	Ext         string       `json:"ext"`
	IsDir       bool         `json:"isDir"`
	Size        int64        `json:"size"`
	SizeStr     string       `json:"sizeStr"`
	ModTime     time.Time    `json:"modTime"`
	IsImage     bool         `json:"isImage"`
	IsText      bool         `json:"isText"`
	IsVideo     bool         `json:"isVideo"`
	Link        string       `json:"link"`
	Src         template.URL `json:"src"`
	TsRank      float64      `db:"ts_rank" json:"tsRank"` // For full-text search ranking
}

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
	Id            uuid.UUID `json:"id"`
	DirectoryId   uuid.UUID `json:"directoryId"`
	Keywords      string    `json:"keywords"`
	SizeSimple    string    `json:"sizeSimple"`
	ModTimeSimple string    `json:"modTimeStr"`
	Type          string    `json:"type"`
	Url           string    `json:"url"`
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

	// go explain(stmt, name, limit, offset)W

	for rows.Next() {

		tsRank := float64(0)
		id := uuid.UUID{}
		data := FinfoJSON{}

		rawData := []byte{}

		err := rows.Scan(
			&id,
			&rawData,
			&tsRank,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawData, &data)

		dataWithId := FileData{
			FinfoJSON: data,
			Id:        id,
		}

		dataWithId.Keywords = name
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
		return err
	}
	defer conn.Close()

	err = conn.QueryRow(stmt, name).Scan(&flist.Count)

	if err != nil {
		return err
	}

	return nil
}

type Directory struct {
	Id        uuid.UUID `db:"id" json:"id"`
	Path      string    `db:"path" json:"path"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
	Done      bool      `db:"scanned" json:"scanned"`
}

type Directries struct {
	List  []Directory       `json:"list"`
	Body  map[string]string `json:"body"`
	Title string            `json:"title"`
}

func (l *Directries) GetList() error {
	const query = `SELECT id, json data FROM directory;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(query)
	if err != nil {
		return err
	}

	for rows.Next() {

		id := uuid.UUID{}
		data := Directory{}

		rawData := []byte{}

		err := rows.Scan(
			&id,
			&rawData,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawData, &data)

		dataWithId := Directory{
			Id:   id,
			Path: data.Path,
		}

		if err != nil {
			return err
		}

		l.List = append(l.List, dataWithId)
	}

	return nil
}

func (f *FileData) SimplifyDetails() {
	f.SizeSimple = utils.ConvertBytes(f.Size)
	f.ModTimeSimple = f.ModTime.Format("2006-01-02 15:04:05")
}

func (f *FileData) CheckExtension() (e error) {

	var url string
	var fileType string

	if slices.Contains(textFiles, f.Ext) {
		fileType = "txt"
		url = fmt.Sprintf("%s/%s.%v", f.Directory, f.Name, f.Ext)
	} else if slices.Contains(imageFiles, f.Ext) {
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
		f.Type = "image"
	} else if slices.Contains(videoFiles, f.Ext) {
		fileType = "video"
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
	}

	f.Type = fileType
	f.Url = url
	return nil
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

	f.SimplifyDetails()

	if err != nil {
		return fmt.Errorf("Error retrieving file by ID:\n%v", err)
	}

	return nil
}

// type FilesDataList struct {
// 	List  []FileData
// 	Count int
// }

// ToJSON converts Finfo to FinfoJSON
func (f *Finfo) ToJSON() FinfoJSON {
	return FinfoJSON{
		Directory: f.Directory,
		Name:      f.Name,
		Ext:       f.Ext,
		IsDir:     f.IsDir,
		Size:      f.Size,
		ModTime:   f.ModTime,
	}
}

type FinfoDetail struct {
	*Finfo
	Title   string
	Preview string
	HTML    template.HTML
}

type IndexedDir struct {
	Id      uuid.UUID `db:"id" json:"id"`
	Name    string    `db:"name" json:"name"`
	Done    bool      `db:"done" json:"done"`
	Created time.Time `db:"created" json:"created"`
	Updated time.Time `db:"updated" json:"updated"`
}

type IndexedDirs struct {
	Indexeddirs []IndexedDir `json:"indexedDirs"`
	Text        string
	HeaderTitle string
	Status      bool
	Params      map[string]string
	Error       map[string]string
}

type User struct {
	Id       uuid.UUID `db:"id" json:"id"`
	Username string    `db:"username" json:"username"`
	Password string    `db:"password" json:"-"`
}

func (p *IndexedDirs) SetParams(c echo.Context) error {

	// Initialize maps if they're nil
	if p.Params == nil {
		p.Params = make(map[string]string)
	}
	if p.Error == nil {
		p.Error = make(map[string]string)
	}

	params := c.FormValue("path")
	p.Params["path"] = utils.CleanInput(params)
	p.Status = true

	return nil
}

func (i *IndexedDirs) List() error {

	const query = `SELECT id, name, done, created FROM directory ORDER BY created DESC;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query indexed directories: %w", err)
	}

	for rows.Next() {
		var dir IndexedDir
		if err := rows.Scan(&dir.Id, &dir.Name, &dir.Done, &dir.Created); err != nil {
			return fmt.Errorf("failed to scan indexed directory: %w", err)
		}

		i.Indexeddirs = append(i.Indexeddirs, dir)
	}

	if len(i.Indexeddirs) == 0 {
		i.Text = "No indexed directories found."
	}

	return nil
}

func (i *IndexedDirs) Append() error {

	query := `INSERT INTO directory (name, done, created) VALUES ($1, $2, $3) RETURNING id, name, done, created;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	newDir := IndexedDir{}
	err = conn.QueryRow(query, i.Params["path"], false, time.Now()).Scan(
		&newDir.Id, &newDir.Name, &newDir.Done, &newDir.Created)
	if err != nil {
		return fmt.Errorf("failed to insert into indexed directories: %w", err)
	}

	// Add the new directory to the slice
	i.Indexeddirs = append(i.Indexeddirs, newDir)

	return nil

}

func (f *Finfo) CheckExtension() error {

	if f.IsDir {
		fSrc := strings.ReplaceAll(f.Directory+"\\"+f.Name, "\\", "/")
		f.Src = template.URL(fSrc)
		return nil
	}

	if slices.Contains(textFiles, f.Ext) {
		f.IsText = true
		f.Link = fmt.Sprintf("%s\\%s.%v", f.Directory, f.Name, f.Ext)

	} else if slices.Contains(imageFiles, f.Ext) {
		f.IsImage = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Directory, f.Name, f.Ext)
	} else if slices.Contains(videoFiles, f.Ext) {
		f.IsVideo = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Directory, f.Name, f.Ext)
	}

	f.Link = strings.ReplaceAll(f.Link, "\\", "/")
	fSrc := fmt.Sprintf("%s\\%s.%v", f.Directory, f.Name, f.Ext)
	fSrc = strings.ReplaceAll(fSrc, "\\", "/")
	f.Src = template.URL(fSrc)

	return nil
}

func (f *Finfo) String() string {
	return fmt.Sprintf("%s, %s, %s, %T\n", f.Directory, f.Name, f.Ext, f.IsDir)
}

type IndexData struct {
	TC          []Finfo `json:"FileList"`
	Text        string
	HeaderTitle string
	Counter     int
	Params      map[string]string
	Error       map[string]string
}

type SearchParams struct {
	Params       string
	Like         string
	Dir          string
	Keywords     string
	Limit        int
	Offset       int
	QueryParam   string
	Stmt         string
	CounterStmt  string
	Ext          string
	Placeholders []any
	Error        map[string]string
}

func (sp *SearchParams) SetParams(c echo.Context) error {

	// Initialize Error map if it's nil
	if sp.Error == nil {
		sp.Error = make(map[string]string)
	}

	method := c.Request().Method

	switch method {

	case http.MethodGet:

		if len(c.QueryParam("name")) == 0 {
			sp.Error["Error"] = "No search parameters provided"
		} else {
			sp.Params = utils.CleanInput(c.QueryParam("name"))

			if len(c.QueryParam("like")) > 0 {
				sp.Like = utils.CleanInput(c.QueryParam("like"))
			}

			if len(c.QueryParam("dir")) > 0 {
				sp.Dir = utils.CleanInput(c.QueryParam("dir"))
			}

			if len(c.QueryParam("keywords")) > 0 {
				sp.Keywords = utils.CleanInput(c.QueryParam("keywords"))
			}

			offsetStr := c.QueryParam("offset")
			limitStr := c.QueryParam("limit")

			if len(limitStr) > 0 {
				sp.Limit, _ = strconv.Atoi(limitStr)
			} else {
				sp.Limit = 10
			}

			if len(offsetStr) > 0 {
				sp.Offset, _ = strconv.Atoi(offsetStr)
			} else {
				sp.Offset = 0
			}
		}

	case http.MethodPost:
		params := c.FormValue("name")
		sp.Params = utils.CleanInput(params)

		if len(c.FormValue("like")) > 0 {
			sp.Like = c.FormValue("like")
		}

		if len(c.FormValue("dir")) > 0 {
			sp.Dir = c.FormValue("dir")
		}

		if len(c.FormValue("keywords")) > 0 {
			sp.Keywords = c.FormValue("keywords")
		}

		sp.Limit = 10
		sp.Offset = 0

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

// NewSearchParams creates a new SearchParams with initialized maps
func NewSearchParams() *SearchParams {
	return &SearchParams{
		Placeholders: make([]any, 0),
		Error:        make(map[string]string),
		Limit:        10,
		Offset:       0,
	}
}

// NewIndexData creates a new IndexData with initialized maps
func NewIndexData() *IndexData {
	return &IndexData{
		TC:     make([]Finfo, 0),
		Params: make(map[string]string),
		Error:  make(map[string]string),
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

func explain(stmt string, placeholders ...any) {
	conn, err := utils.PgConn()
	if err != nil {
		log.Println("Error connecting to the database for search words:", err)
		return
	}
	defer conn.Close()
	explainAnalyze := fmt.Sprintf(`EXPLAIN ANALYZE %s`, stmt)
	fmt.Println()
	fmt.Printf("%v %v", explainAnalyze, placeholders)
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
	if err := explainRows.Err(); err != nil {
		fmt.Println("Error iterating EXPLAIN ANALYZE rows:", err)
	}
}
