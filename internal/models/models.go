package models

import (
	"fmt"
	"gofiles/internal/utils"
	"html/template"
	"net/http"
	"regexp"
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
	Id          uuid.UUID    `db:"id" json:"id"`
	DirectoryId uuid.UUID    `db:"directory_id" json:"directoryId"`
	Directory   string       `db:"directory" json:"directory"`
	Name        string       `db:"name" json:"name"`
	Ext         string       `db:"ext" json:"ext"`
	IsDir       bool         `db:"is_dir" json:"isDir"`
	Size        int64        `db:"size" json:"size"`
	SizeStr     string       `json:"sizeStr"`
	ModTime     time.Time    `db:"modTime"`
	IsImage     bool         `json:"isImage"`
	IsText      bool         `json:"isText"`
	IsVideo     bool         `json:"isVideo"`
	Link        string       `json:"link"`
	Src         template.URL `json:"src"`
	TsRank      float64      `db:"ts_rank" json:"tsRank"` // For full-text search ranking
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

	query := `SELECT id, name, done, created FROM directory ORDER BY created DESC;`

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

func (s *SearchParams) QueryStmt() error {

	switcher := 0
	isOn := []string{"on", "true", "1", "yes", "ok", "y", "tak", "t"}

	if slices.Contains(isOn, s.Like) {
		switcher += 1
	}

	if slices.Contains(isOn, s.Dir) {
		switcher += 100
	}

	language := "'polish'"
	if len(s.Keywords) > 0 {
		language = "'english'"
	}

	s.QueryParam = s.Params

	if strings.Contains(s.Params, ".") {
		// Regex: split on dot, comma, semicolon, or any whitespace
		re := regexp.MustCompile(`[.,;]+`)
		parts := re.Split(s.Params, -1)
		s.QueryParam = parts[0]
		s.QueryParam = strings.TrimSpace(s.QueryParam)
		s.Ext = parts[1]
	}

	switch switcher {
	case 0:

		// column = keywords
		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, directory, size, mod_time,
		ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
		FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
		ORDER BY ts_rank DESC
		LIMIT $2 OFFSET $3;`, language)

		s.CounterStmt = fmt.Sprintf(`
		SELECT COUNT(id) FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, language)

	case 1:
		s.QueryParam = "^" + s.QueryParam
		// column = "name"
		s.Stmt = `
		SELECT id, name, ext, is_dir, directory, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1)
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`

		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1);`
	case 100:

		// column = "files.keywords"

		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, directory, size, mod_time FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, language)

		s.CounterStmt = fmt.Sprintf(`
		SELECT COUNT(*) FROM files 
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND is_dir=true;`, language)

	case 101:
		s.QueryParam = "^" + s.QueryParam

		s.Stmt = `
		SELECT id, name, ext, is_dir, directory, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1) AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`
		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1) AND is_dir=true;`
	}

	s.Placeholders = []any{s.QueryParam, s.Limit, s.Offset}

	if len(s.Ext) > 0 {

		s.Stmt = fmt.Sprintf(`SELECT files.id, name, files.ext, is_dir, directory, size, mod_time,
		ts_rank( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
		FROM files
		JOIN ext ON files.ext_id = ext.id
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND ext.ext = $2
		ORDER BY ts_rank DESC LIMIT $3 OFFSET $4;`, language)

		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files
		JOIN ext ON files.ext_id = ext.id
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND ext.ext = $2;`, language)
		s.Placeholders = []any{s.Placeholders[0], s.Ext, s.Placeholders[1], s.Placeholders[2]}
	}

	return nil

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
