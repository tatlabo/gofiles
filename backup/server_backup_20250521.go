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
	"slices"
	"strconv"
	"strings"
	"time"

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

	err = checkExtensionItem(&item)
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

	err = checkExtensionItem(&finfo)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error checking extension item")
	}

	var ctx FinfoDetail
	ctx.Finfo = finfo

	if ctx.IsText == true {
		ctx.HTML, _ = textToChoroma(finfo)
	}

	return c.String(http.StatusOK, string(ctx.HTML))
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

	err = checkExtensionItem(&finfo)
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

type SearchParams struct {
	Params      []string
	Like        string
	Dir         string
	Limit       int
	Offset      int
	QueryParam  string
	Stmt        string
	CounterStmt string
	Ext         string
}

func searchParams(c echo.Context) (fp SearchParams) {
	method := c.Request().Method

	fp = SearchParams{}

	substring := "."

	switch method {
	case http.MethodPost:
		params := c.FormValue("name")
		params = strings.ToLower(params)
		paramsList := []string{}

		if strings.Contains(params, ".") {
			paramsList = strings.Split(params, substring)
			fp.Params = append(fp.Params, paramsList...)
		} else {
			fp.Params = append(fp.Params, params)
		}

		if len(c.FormValue("like")) > 0 {
			fp.Like = c.FormValue("like")
		}

		if len(c.FormValue("dir")) > 0 {
			fp.Dir = c.FormValue("dir")
		}

		fp.Limit = 10
		fp.Offset = 0

	case http.MethodGet:

		if len(c.QueryParam("like")) > 0 {
			fp.Like = c.QueryParam("like")
		}

		if len(c.QueryParam("dir")) > 0 {
			fp.Dir = c.QueryParam("dir")
		}

		fp.Params = append(fp.Params, c.QueryParam("name"))
		offsetStr := c.QueryParam("offset")
		limitStr := c.QueryParam("limit")

		if len(limitStr) > 0 {
			fp.Limit, _ = strconv.Atoi(limitStr)
		} else {
			fp.Limit = 10
		}

		if len(offsetStr) > 0 {
			fp.Offset, _ = strconv.Atoi(offsetStr)
		} else {
			fp.Offset = 0
		}

	}

	return fp
}

func queryStmt(sp *SearchParams) error {

	switcher := 0
	if sp.Like == "on" {
		switcher += 1
	}
	if sp.Dir == "on" {
		switcher += 10
	}

	clause := []string{`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE LOWER(name)`, "", `ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`}
	counterClause := `SELECT COUNT(*) FROM files WHERE LOWER(name)`
	sp.QueryParam = strings.Join(sp.Params, ".")

	switch switcher {
	case 0:
		clause[1] = "=$1"
	case 1:
		clause[1] = "LIKE $1 AND is_dir=false"
		sp.QueryParam = "%" + sp.QueryParam + "%"
	case 10:
		clause[1] = "LIKE $1 AND is_dir=true"
	case 11:
		sp.QueryParam = "%" + sp.QueryParam + "%"
		clause[1] = "LIKE $1 AND is_dir=true"
	}

	sp.Stmt = strings.Join(clause[:], " ")
	sp.CounterStmt = strings.Join([]string{counterClause, clause[1], ";"}, " ")
	return nil
}

func findInDb(c echo.Context) (IndexData, error) {

	indexData := IndexData{}

	searchParams := searchParams(c)

	_ = queryStmt(&searchParams)

	log.Printf("Query: %#v", searchParams)

	conn, err := utils.PgConn()
	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error connecting to the database")
	}
	defer conn.Close()

	rows, err := conn.Query(searchParams.Stmt, searchParams.QueryParam, searchParams.Limit, searchParams.Offset)
	log.Println(searchParams.Stmt, searchParams.Params, searchParams.Limit, searchParams.Offset)
	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error executing query")
	}

	var counter int
	err = conn.QueryRow(searchParams.CounterStmt, searchParams.QueryParam).Scan(&counter)
	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error executing count query")
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

		err = checkExtensionItem(&finfo)
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
	context.HeaderTitle = "Founded in db: " + strings.Join(searchParams.Params, ".")
	context.Text = "Content preview"
	context.Counter = counter
	context.Params = make(map[string]string)
	context.Params["Name"] = strings.Join(searchParams.Params, ".")
	context.Params["Offset"] = strconv.Itoa(searchParams.Offset)
	context.Params["Limit"] = strconv.Itoa(searchParams.Limit)
	context.Params["Like"] = searchParams.Like
	context.Params["Dir"] = searchParams.Dir
	if searchParams.Offset+searchParams.Limit >= counter {
		context.Params["NextPage"] = ""
	} else {
		context.Params["NextPage"] = fmt.Sprintf("/json/search?name=%s&offset=%d&limit=%d", strings.Join(searchParams.Params, "."), searchParams.Offset+searchParams.Limit, searchParams.Limit)
	}
	context.Params["PreviousPage"] = fmt.Sprintf("")

	// log.Println(context.Params)

	return context, nil
}

func checkExtensionItem(f *Finfo) error {

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

func searchInDb(c echo.Context) error {
	context, err := findInDb(c)

	if err != nil {
		return c.String(http.StatusInternalServerError, "Error finding in db"+fmt.Sprint(err))
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

func FindForm(c echo.Context) error {
	return c.Render(http.StatusOK, "search", "")
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

	e.GET("/", FindForm)

	e.GET("/detail/:id", detailById)
	e.GET("/details/:id", detailById)
	e.GET("/preview/:id", previewById)
	e.GET("/preview/image/:id", previewImage)

	e.GET("/search", FindForm)
	e.POST("/search", searchInDb)
	// e.GET("/files", findInDb)

	e.GET("/json/search", searchJson)

	e.GET("/append", searchAppend)

	e.Logger.Fatal(e.Start(":8000"))

}
