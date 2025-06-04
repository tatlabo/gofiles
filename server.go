package main

import (
	"fmt"
	"gofiles/chroma"
	"gofiles/utils"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/labstack/echo/v4"
)

type Finfo struct {
	Id       int       `json:"id"`
	ParentId int       `json:"parentId"`
	Path     string    `json:"path"`
	Name     string    `json:"name"`
	Ext      string    `json:"ext"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
	SizeStr  string    `json:"sizeStr"`
	ModTime  time.Time `json:"modTime"`
	IsImage  bool
	IsText   bool
	IsVideo  bool
	Link     string
	Src      template.URL
	Empty    bool
}

type FinfoDetail struct {
	Finfo
	Title   string
	Preview string
	HTML    template.HTML
}

type IndexData struct {
	TC          []Finfo `json:"FileList"`
	Text        string
	HeaderTitle string
	Counter     int
	Params      map[string]string
	Error       map[string]string
}

var limit int

var textFiles = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv"}
var imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tiff", "webp", "svg", "ico", "heic", "raw"}
var videoFiles = []string{"mp4"}

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05") // Customize the format as needed
}

func not(b bool) bool {
	return !b
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

func (sp *SearchParams) SearchParams(c echo.Context) error {

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

		sp.Limit = 10
		sp.Offset = 0

	}

	return nil
}

type SearchParams struct {
	Params         string
	Like           string
	Dir            string
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
		re := regexp.MustCompile(`[.,;\s]+`)
		parts := re.Split(s.Params, -1)
		s.QueryParam = parts[0]
		s.Ext = parts[1]
	}

	clause := ""

	switch switcher {
	case 0:
		clause = "=$1"
	case 1:
		s.QueryParam = s.QueryParam + "%"
		clause = "LIKE $1"
	case 100:
		clause = "LIKE $1 AND is_dir=true"
	case 101:
		s.QueryParam = "%" + s.QueryParam + "%"
		clause = "LIKE $1 AND is_dir=true"
	}

	s.Stmt = fmt.Sprintf(`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE LOWER(name) %s ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, clause)
	s.ExplainAnalyze = fmt.Sprintf(`EXPLAIN ANALYZE %s`, s.Stmt)

	s.Placeholders = []any{s.QueryParam, s.Limit, s.Offset}

	s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files WHERE LOWER(name) %s;`, clause)
	if len(s.Ext) > 0 {
		s.Stmt = fmt.Sprintf(`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE LOWER(name) %s AND LOWER(ext) = $2 ORDER BY mod_time DESC LIMIT $3 OFFSET $4;`, clause)
		s.ExplainAnalyze = fmt.Sprintf(`EXPLAIN ANALYZE %s`, s.Stmt)
		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files WHERE LOWER(name) %s AND LOWER(ext) = $2;`, clause)
		s.Placeholders = []any{s.Placeholders[0], s.Ext, s.Placeholders[1], s.Placeholders[2]}
	}

	return nil

}

func findInDb(c echo.Context) (IndexData, error) {

	indexData := IndexData{}

	var searchParams SearchParams
	err := searchParams.SearchParams(c)

	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error parsing search parameters")
	}

	if searchParams.Error["Error"] != "" {
		indexData.Error = searchParams.Error
		return indexData, nil
	}

	err = searchParams.QueryStmt()
	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error creating query statement")
	}

	conn, err := utils.PgConn()
	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error connecting to the database")
	}
	defer conn.Close()

	var counter int
	// var explainAnalyze string

	rows, err := conn.Query(searchParams.Stmt, searchParams.Placeholders...)

	if err != nil {
		log.Println("Error running EXPLAIN ANALYZE:", err)
		// handle error
	}

	// explainRows, err := conn.Query(searchParams.ExplainAnalyze, searchParams.Placeholders...)
	// for explainRows.Next() {
	// 	var line string
	// 	if err := explainRows.Scan(&line); err != nil {
	// 		log.Println("Error scanning EXPLAIN ANALYZE line:", err)
	// 		continue
	// 	}
	// 	log.Println(line)
	// }
	// if err := explainRows.Err(); err != nil {
	// 	log.Println("Error iterating EXPLAIN ANALYZE rows:", err)
	// }

	if err != nil {
		c.Render(200, "error", err)
		return indexData, err
	}

	if searchParams.Ext != "" {
		err = conn.QueryRow(searchParams.CounterStmt, searchParams.QueryParam, searchParams.Ext).Scan(&counter)
	} else {
		err = conn.QueryRow(searchParams.CounterStmt, searchParams.QueryParam).Scan(&counter)
	}
	if err != nil {
		log.Println("Error executing count query:", err)
	}

	fInfoList := []Finfo{}

	for rows.Next() {
		var finfo = Finfo{}
		// Scan the current row into variables
		err := rows.Scan(
			&finfo.Id,
			&finfo.Name,
			&finfo.Ext,
			&finfo.IsDir,
			&finfo.Path,
			&finfo.Size,
			&finfo.ModTime,
		)

		// Check for errors during scanning

		if err != nil {
			return indexData, c.String(http.StatusInternalServerError, "Error scanning result")
		}

		if err := rows.Err(); err != nil {
			return indexData, c.String(http.StatusInternalServerError, "Error iterating over rows")
		}

		finfo.Path = strings.ReplaceAll(finfo.Path, "\\", "/")
		finfo.SizeStr = utils.ConvertBytes(finfo.Size)

		_ = finfo.CheckExtension()
		if err != nil {
			return indexData, c.String(http.StatusInternalServerError, "Error checking extension item")
		}

		if finfo.IsImage {
			finfo.Src = template.URL(finfo.Link)
		}

		fInfoList = append(fInfoList, finfo)
	}

	context := IndexData{}
	context.TC = fInfoList
	context.HeaderTitle = "Founded in db: " + searchParams.Params
	context.Text = "Content preview"
	context.Counter = counter
	context.Params = map[string]string{
		"Name":         searchParams.Params,
		"Like":         searchParams.Like,
		"Dir":          searchParams.Dir,
		"Limit":        strconv.Itoa(searchParams.Limit),
		"Offset":       strconv.Itoa(searchParams.Offset),
		"NextPage":     "",
		"PreviousPage": "",
	}

	if searchParams.Offset+searchParams.Limit < counter {
		context.Params["NextPage"] = fmt.Sprintf("/json/search?name=%s&offset=%d&limit=%d", searchParams.Params, searchParams.Offset+searchParams.Limit, searchParams.Limit)
	}
	context.Params["PreviousPage"] = fmt.Sprintf("")

	return context, nil
}

func searchInDb(c echo.Context) error {
	context, err := findInDb(c)

	if context.Error["Error"] != "" {
		c.Render(http.StatusOK, "error", context)
		return nil
	}

	if err != nil {
		if context.Error == nil {
			context.Error = make(map[string]string)
		}
		context.Error["Error"] = "Error finding in db, table doesn't exist: " + fmt.Sprint(err)
		return c.Render(http.StatusOK, "error", context)
	}

	if len(context.TC) == 0 {
		context.Params["NotFound"] = "true"
		err = c.Render(http.StatusOK, "index", context)
	} else {
		err = c.Render(http.StatusOK, "index", context)
	}
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error rendering template"+fmt.Sprint(err))
	}
	return nil
}

func searchAppend(c echo.Context) error {
	context, err := findInDb(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error finding in db"+fmt.Sprint(err))
	}
	c.Render(http.StatusOK, "list", context)
	return nil
}

func searchJson(c echo.Context) error {
	context, err := findInDb(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error finding in db"+fmt.Sprint(err))
	}
	c.JSON(http.StatusOK, context)
	return nil
}

func startpage(c echo.Context) error {
	c.Render(http.StatusOK, "startpage", nil)
	return nil
}

func equals(a, b interface{}) bool {
	return a == b
}

func main() {

	// Initialize Echo framework
	e := echo.New()

	t := &Template{
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"formatDate": formatDate, // Register the custom function
			"not":        not,
			"equals":     equals,
		}).ParseGlob("public/views/*.html")),
	}

	e.Static("/static", "static")
	e.Static("/media", "media")
	e.Renderer = t

	e.GET("/", startpage)
	e.GET("/search", searchInDb) // FindForm
	e.POST("/search", searchInDb)
	e.GET("/detail/:id", detailById)
	e.GET("/details/:id", detailById)
	e.GET("/preview/:id", previewById)
	e.GET("/preview/image/:id", previewImage)

	// e.GET("/files", findInDb)

	e.GET("/json/search", searchJson)

	e.GET("/append", searchAppend)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	e.Logger.Fatal(e.Start(":8000"))

}

func previewImage(c echo.Context) error {

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}

	conn, err := utils.PgConn()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error connecting to the database")
	}
	defer conn.Close()

	finfo := Finfo{}
	stmt := fmt.Sprintf(`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE id = $1;`)

	err = conn.QueryRow(stmt, id).Scan(
		&finfo.Id,
		&finfo.Name,
		&finfo.Ext,
		&finfo.IsDir,
		&finfo.Path,
		&finfo.Size,
		&finfo.ModTime,
	)

	if err != nil {
		return c.String(http.StatusInternalServerError, "Error executing query")
	}

	srcPath := fmt.Sprintf("%s/%s.%v", finfo.Path, finfo.Name, finfo.Ext)
	srcPath = strings.ReplaceAll(srcPath, "\\", "/")
	destPath := fmt.Sprintf("media/images/%s.%v", finfo.Name, finfo.Ext)
	destPath = strings.ReplaceAll(destPath, "\\", "/")

	_ = finfo.CheckExtension()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking extension item")
	}

	// Ensure the destination directory exists
	if err := os.MkdirAll("media/images", os.ModePerm); err != nil {
		return c.String(http.StatusInternalServerError, "Error creating destination directory")
	}

	// Copy the image file
	if err := copyImageFile(srcPath, destPath); err != nil {
		log.Printf("Error copying file from %s to %s: %v", srcPath, destPath, err)
		return c.String(http.StatusInternalServerError, "Error copying image file")
	}

	go func(filePath string) {
		time.Sleep(120 * time.Second)
		if err := os.Remove(filePath); err != nil {
			log.Printf("Error deleting file %s: %v", filePath, err)
		} else {
			log.Printf("File %s deleted successfully", filePath)
		}
	}(destPath)

	finfo.Src = template.URL(destPath)

	return c.Render(http.StatusOK, "single_page", finfo)
}

func copyImageFile(srcPath, destPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Ensure the destination file is properly written to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

func detailById(c echo.Context) error {

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}

	conn, err := utils.PgConn()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error connecting to the database")
	}
	defer conn.Close()

	item := Finfo{}
	stmt := fmt.Sprintf(`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE id = $1;`)

	err = conn.QueryRow(stmt, id).Scan(
		&item.Id,
		&item.Name,
		&item.Ext,
		&item.IsDir,
		&item.Path,
		&item.Size,
		&item.ModTime,
	)

	if err != nil {
		return c.String(http.StatusInternalServerError, "Error executing query")
	}

	_ = item.CheckExtension()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking extension item")
	}

	fc := FinfoDetail{}
	fc.Finfo = item
	fc.Title = fmt.Sprintf("Details for: %s.%s", item.Name, item.Ext)

	if fc.IsText == true {
		fc.HTML, _ = textToChoroma(item)
	}

	return c.Render(http.StatusOK, "detail", fc)
}

func previewById(c echo.Context) error {

	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid ID")
	}

	conn, err := utils.PgConn()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error connecting to the database")
	}
	defer conn.Close()

	finfo := Finfo{}
	stmt := fmt.Sprintf(`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE id = $1;`)

	err = conn.QueryRow(stmt, id).Scan(
		&finfo.Id,
		&finfo.Name,
		&finfo.Ext,
		&finfo.IsDir,
		&finfo.Path,
		&finfo.Size,
		&finfo.ModTime,
	)

	if err != nil {
		return c.String(http.StatusInternalServerError, "Error executing query")
	}

	err = finfo.CheckExtension()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking extension item")
	}

	var ctx FinfoDetail
	ctx.Finfo = finfo

	if ctx.IsText == true {
		ctx.HTML, _ = textToChoroma(finfo)
	}

	wrap := fmt.Sprintf("<div><h3>%s.%s</h3><p>%s</p>%s</div>", ctx.Finfo.Name, ctx.Finfo.Ext, ctx.Finfo.Path, string(ctx.HTML))

	return c.String(http.StatusOK, wrap)
}

func textToChoroma(f Finfo) (template.HTML, error) {

	address := fmt.Sprintf("%s\\%s.%v", f.Path, f.Name, f.Ext)
	fin, err := os.Open(address)

	if err != nil {
		return "", err
	}
	defer fin.Close()

	highlightCode, err := chroma.HighlightCode(address)

	if err != nil {
		return "", err
	}

	return template.HTML(highlightCode), nil

}
