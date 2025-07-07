package main

import (
	"fmt"
	"gofiles/chroma"
	"gofiles/models"
	"gofiles/utils"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "net/http/pprof"

	"github.com/labstack/echo/v4"
)

var limit int

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

func findInDb(c echo.Context) (models.IndexData, error) {

	indexData := models.IndexData{}

	var searchParams models.SearchParams
	err := searchParams.SetParams(c)

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

	log.Println(searchParams.Stmt, searchParams.Placeholders)
	explainRows, err := conn.Query(searchParams.ExplainAnalyze, searchParams.Placeholders...)
	for explainRows.Next() {
		var line string
		if err := explainRows.Scan(&line); err != nil {
			log.Println("Error scanning EXPLAIN ANALYZE line:", err)
			continue
		}
		log.Println(line)
	}
	if err := explainRows.Err(); err != nil {
		log.Println("Error iterating EXPLAIN ANALYZE rows:", err)
	}

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

	fInfoList := []models.Finfo{}

	for rows.Next() {
		var finfo = models.Finfo{}
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

		if finfo.IsImage {
			finfo.Src = template.URL(finfo.Link)
		}

		fInfoList = append(fInfoList, finfo)
	}

	indexData.TC = fInfoList
	indexData.HeaderTitle = "Founded in db: " + searchParams.Params
	indexData.Text = "Content preview"
	indexData.Counter = counter
	indexData.Params = map[string]string{
		"Name":         searchParams.Params,
		"Like":         searchParams.Like,
		"Dir":          searchParams.Dir,
		"Keywords":     searchParams.Keywords,
		"Limit":        strconv.Itoa(searchParams.Limit),
		"Offset":       strconv.Itoa(searchParams.Offset),
		"NextPage":     "",
		"PreviousPage": "",
	}

	if searchParams.Offset+searchParams.Limit < counter {
		indexData.Params["NextPage"] = fmt.Sprintf("/json/search?name=%s&offset=%d&limit=%d", searchParams.Params, searchParams.Offset+searchParams.Limit, searchParams.Limit)
	}
	indexData.Params["PreviousPage"] = ""

	return indexData, nil
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

	e.Logger.Fatal(e.Start(":8"))

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

	finfo := models.Finfo{}
	stmt := (`SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE id = $1;`)

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

	item := models.Finfo{}
	stmt := `SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE id = $1;`

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

	item.SizeStr = utils.ConvertBytes(item.Size)

	item.Title = fmt.Sprintf("Details for: %s.%s", item.Name, item.Ext)
	if item.IsText == true {
		item.HTML, _ = textToChoroma(item)
	}

	return c.Render(http.StatusOK, "detail", item)
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

	finfo := models.Finfo{}
	stmt := `SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE id = $1;`

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

	if finfo.IsText == true {
		finfo.HTML, _ = textToChoroma(finfo)
	}

	wrap := fmt.Sprintf("<div><h3>%s.%s</h3><p>%s</p>%s</div>", finfo.Name, finfo.Ext, finfo.Path, string(finfo.HTML))

	return c.String(http.StatusOK, wrap)
}

func textToChoroma(f models.Finfo) (template.HTML, error) {

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
