package models

import (
	"fmt"
	"gofiles/utils"
	"html/template"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var textFiles = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv"}
var imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tiff", "webp", "svg", "ico", "heic", "raw"}
var videoFiles = []string{"mp4"}

type Finfo struct {
	Id       int    `db:"id" json:"id"`
	ParentId int    `db:"parent_id" json:"parent_id"`
	Path     string `db:"path" json:"path"`
	Name     string `db:"name" json:"name"`
	Ext      string `db:"ext" json:"ext"`
	IsDir    bool   `db:"is_dir" json:"is_dir"`
	Size     int64  `db:"size"`
	SizeStr  string
	ModTime  time.Time `db:"mod_time"`
	IsImage  bool
	IsText   bool
	IsVideo  bool
	Link     string
	Src      template.URL
	Empty    bool

	Title   string
	Preview string
	HTML    template.HTML
}

func (f *Finfo) CheckExtension() error {

	if slices.Contains(textFiles, f.Ext) {
		f.IsText = true
		f.Link = fmt.Sprintf("%s\\%s.%v", f.Path, f.Name, f.Ext)
		f.Link = strings.ReplaceAll(f.Link, "\\", "/")

	} else if slices.Contains(imageFiles, f.Ext) {
		f.IsImage = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Path, f.Name, f.Ext)
	} else if slices.Contains(videoFiles, f.Ext) {
		f.IsVideo = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Path, f.Name, f.Ext)
	}

	fSrc := fmt.Sprintf("%s\\%s.%v", f.Path, f.Name, f.Ext)
	fSrc = strings.ReplaceAll(fSrc, "\\", "/")
	f.Src = template.URL(fSrc)

	return nil
}

func (f *Finfo) String() string {
	return fmt.Sprintf("%s, %s, %s, %T\n", f.Path, f.Name, f.Ext, f.IsDir)
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
	Params         string
	Like           string
	Dir            string
	Keywords       string
	Limit          int
	Offset         int
	QueryParam     string
	Stmt           string
	ExplainAnalyze string
	CounterStmt    string
	Ext            string
	Placeholders   []any
	Error          map[string]string
}

func (s *SearchParams) QueryStmt() error {

	switcher := 0
	isOn := []string{"on", "true", "1", "yes", "ok", "y", "tak", "t"}

	if slices.Contains(isOn, s.Like) {
		switcher += 1
	}

	if len(s.Ext) > 0 {
		s.QueryParam = "%" + s.QueryParam + "%"
		switcher += 10
	}

	if slices.Contains(isOn, s.Dir) {
		switcher += 100
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

	clause := ""

	column := "files.keywords"
	tableName := "files"

	switch switcher {
	case 0:
		clause = s.QueryParam

		s.Stmt = `
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE websearch_to_tsquery('polish', $1) @@ to_tsvector('polish', keywords)
		ORDER BY mod_time DESC
		LIMIT $2 OFFSET $3;`

		s.CounterStmt = `
		SELECT COUNT(id) FROM files
		WHERE websearch_to_tsquery('polish', $1) @@ to_tsvector('polish', keywords);`

	case 1:
		s.QueryParam = s.QueryParam + "%"
		clause = column + " LIKE $1"

		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE %s 
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, clause)

		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files WHERE %s;`, clause)
	case 100:
		clause = s.Params

		s.Stmt = fmt.Sprintln(`
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE websearch_to_tsquery('polish', $1) @@ to_tsvector('polish', files.keywords)
		AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`)

		s.CounterStmt = fmt.Sprintln(`
		SELECT COUNT(*) FROM files 
		WHERE websearch_to_tsquery('polish', $1) @@ to_tsvector('polish', files.keywords)
		AND is_dir=true;`)

	case 101:
		s.QueryParam = s.QueryParam + "%"
		clause = column + " LIKE $1 AND is_dir=true"

		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, path, size, mod_time FROM %s
		WHERE %s 
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, tableName, clause)

		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files WHERE %s;`, clause)
	}

	// SQL

	// s.Stmt = fmt.Sprintf(`
	// SELECT id, name, ext, is_dir, path, size, mod_time FROM %s
	// WHERE %s
	// ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, tableName, clause)

	s.Placeholders = []any{s.QueryParam, s.Limit, s.Offset}

	if len(s.Ext) > 0 {
		clause = column + " = $1"
		s.Stmt = fmt.Sprintf(`SELECT files.id, name, files.ext, is_dir, path, size, mod_time FROM files
		JOIN ext ON files.ext_id = ext.id 
		-- WHERE %s 
		WHERE websearch_to_tsquery('polish', $1) @@ to_tsvector('polish', files.keywords)
		AND ext.ext = $2
		ORDER BY files.mod_time DESC LIMIT $3 OFFSET $4;`, clause)

		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files 
		JOIN ext ON files.ext_id = ext.id 
		-- WHERE %s 
		WHERE websearch_to_tsquery('polish', $1) @@ to_tsvector('polish', files.keywords)
		AND ext.ext = $2;`, clause)
		s.Placeholders = []any{s.Placeholders[0], s.Ext, s.Placeholders[1], s.Placeholders[2]}
	}

	s.ExplainAnalyze = fmt.Sprintf(`EXPLAIN ANALYZE %s`, s.Stmt)

	return nil

}

func (sp *SearchParams) SetParams(c echo.Context) error {

	method := c.Request().Method

	switch method {

	case http.MethodGet:

		if len(c.QueryParam("name")) == 0 {
			sp.Error = map[string]string{"Error": "No search parameters provided"}
		} else {
			sp.Params = utils.CleanInput(c.QueryParam("name"))

			if len(c.QueryParam("like")) > 0 {
				sp.Like = utils.CleanInput(c.QueryParam("like"))
			}

			if len(c.QueryParam("dir")) > 0 {
				sp.Dir = utils.CleanInput(c.QueryParam("dir"))
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
