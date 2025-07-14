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

	"github.com/labstack/echo/v4"
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Product struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

var textFiles = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv", "R", "r"}
var imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "svg", "ico", "heic", "raw"}
var videoFiles = []string{"mp4", "wav", "mp3", "aif", "aiff"}

type Finfo struct {
	Id       int          `db:"id" json:"id"`
	ParentId int          `db:"parent_id" json:"parentId"`
	Path     string       `db:"path" json:"path"`
	Name     string       `db:"name" json:"name"`
	Ext      string       `db:"ext" json:"ext"`
	IsDir    bool         `db:"is_dir" json:"isDir"`
	Size     int64        `db:"size" json:"size"`
	SizeStr  string       `json:"sizeStr"`
	ModTime  time.Time    `db:"modTime"`
	IsImage  bool         `json:"isImage"`
	IsText   bool         `json:"isText"`
	IsVideo  bool         `json:"isVideo"`
	Link     string       `json:"link"`
	Src      template.URL `json:"src"`
	TsRank   float64      `db:"ts_rank" json:"tsRank"` // For full-text search ranking
}

type FinfoDetail struct {
	*Finfo
	Title   string
	Preview string
	HTML    template.HTML
}

func (f *Finfo) CheckExtension() error {

	if f.IsDir {
		fSrc := strings.ReplaceAll(f.Path+"\\"+f.Name, "\\", "/")
		f.Src = template.URL(fSrc)
		return nil
	}

	if slices.Contains(textFiles, f.Ext) {
		f.IsText = true
		f.Link = fmt.Sprintf("%s\\%s.%v", f.Path, f.Name, f.Ext)

	} else if slices.Contains(imageFiles, f.Ext) {
		f.IsImage = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Path, f.Name, f.Ext)
	} else if slices.Contains(videoFiles, f.Ext) {
		f.IsVideo = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Path, f.Name, f.Ext)
	}

	f.Link = strings.ReplaceAll(f.Link, "\\", "/")
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
		SELECT id, name, ext, is_dir, path, size, mod_time,
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
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1)
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`

		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1);`
	case 100:

		// column = "files.keywords"

		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
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
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1) AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`
		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1) AND is_dir=true;`
	}

	s.Placeholders = []any{s.QueryParam, s.Limit, s.Offset}

	if len(s.Ext) > 0 {

		s.Stmt = fmt.Sprintf(`SELECT files.id, name, files.ext, is_dir, path, size, mod_time,
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

	// s.ExplainAnalyze = fmt.Sprintf(`EXPLAIN ANALYZE %s`, s.Stmt)

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
