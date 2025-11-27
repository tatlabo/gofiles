package handlers

import (
	"fmt"
	"gofiles/internal/models"
	"gofiles/utils"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

func IndexPage(w http.ResponseWriter, r *http.Request) error {
	fmt.Fprintf(w, "Welcome to the file search application!")
	return nil
}

func StartPage(c echo.Context) error {
	c.Render(http.StatusOK, "startpage", nil)
	return nil
}

func SearchInDb(c echo.Context) error {
	context, err := findInDb(c)

	if context.Error["Error"] == "No search parameters provided" {
		context.Params = map[string]string{
			"NotFound":  "true",
			"StartPage": "true",
		}
		c.Render(http.StatusOK, "index", context)
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

func findInDb(c echo.Context) (models.IndexData, error) {

	indexData := models.IndexData{}

	var searchParams models.SearchParams
	err := searchParams.SetParams(c)

	if err != nil {
		return indexData, c.String(http.StatusInternalServerError, "Error parsing search parameters")
	}

	if searchParams.Error["Error"] == "No search parameters provided" {
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

	rows, err := conn.Query(searchParams.Stmt, searchParams.Placeholders...)

	log.Print(searchParams.Stmt, searchParams.Placeholders)

	if err != nil {
		log.Println("Error running EXPLAIN ANALYZE:", err)
		// handle error
	}

	log.Println(searchParams.Stmt, searchParams.Placeholders)

	go func() {
		explainAnalyze(searchParams.Stmt, searchParams.Placeholders...)
	}()

	go func() {
		insertSearchedWord(searchParams.Params)
	}()

	counter := 0
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
		if searchParams.Like == "" && searchParams.Dir == "" {
			err := rows.Scan(
				&finfo.Id,
				&finfo.Name,
				&finfo.Ext,
				&finfo.IsDir,
				&finfo.Directory,
				&finfo.Size,
				&finfo.ModTime,
				&finfo.TsRank,
			)

			if err != nil {
				return indexData, c.String(http.StatusInternalServerError, err.Error())
			}
		} else {
			err := rows.Scan(
				&finfo.Id,
				&finfo.Name,
				&finfo.Ext,
				&finfo.IsDir,
				&finfo.Directory,
				&finfo.Size,
				&finfo.ModTime,
			)

			if err != nil {
				return indexData, c.String(http.StatusInternalServerError, err.Error())
			}
		}
		// Scan the current row into variables

		// Check for errors during scanning

		if err := rows.Err(); err != nil {
			return indexData, c.String(http.StatusInternalServerError, "Error iterating over rows")
		}

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

func ResponseAppend(c echo.Context) error {
	context, err := findInDb(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error finding in db"+fmt.Sprint(err))
	}
	c.Render(http.StatusOK, "list", context)
	return nil
}

func ResponseJson(c echo.Context) error {
	context, err := findInDb(c)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error finding in db"+fmt.Sprint(err))
	}
	c.JSON(http.StatusOK, context)
	return nil
}

func PreviewImage(c echo.Context) error {

	id := c.Param("id")

	finfo, err := SelectFinfoById(id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error executing query by id in\n PreviewImage "+err.Error())
	}

	srcPath := fmt.Sprintf("%s/%s.%v", finfo.Directory, finfo.Name, finfo.Ext)
	destPath := fmt.Sprintf("media/images/%s.%v", finfo.Name, finfo.Ext)

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

func DetailById(c echo.Context) error {

	id := c.Param("id")

	finfo, err := SelectFinfoById(id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error executing query by id in\n DetailById "+err.Error())
	}

	finfo.SizeStr = utils.ConvertBytes(finfo.Size)

	finfodetail := models.FinfoDetail{
		Finfo: &finfo,
	}

	finfodetail.Title = "Details for: " + finfo.Name + finfo.Ext
	if finfo.IsText == true {
		finfodetail.HTML, _ = TxtToChoroma(finfo)
	}

	return c.Render(http.StatusOK, "detail", finfodetail)
}

// DeleteIndexedDirectory handles DELETE requests to remove indexed directories
func DropIndexedDirectory(c echo.Context) error {
	// Get the ID from URL parameter
	id := c.Param("id")

	// Create IndexedDirs instance and delete
	indexedDirs := models.NewIndexedDirs()
	err := indexedDirs.Delete(id)
	if err != nil {
		if err.Error() == fmt.Sprintf("no directory found with ID %v", id) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "Directory not found",
			})
		}

		log.Printf("Error deleting directory with ID %v: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete directory",
		})
	}

	// Return success response
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Directory deleted successfully",
	})
}

// DeleteIndexedDirectory handles DELETE requests to remove indexed directories
func DeleteDirectory(c echo.Context) error {

	id := c.Param("id")

	query := `DELETE FROM directory WHERE id = $1;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Exec(query, id)
	fmt.Printf(`%v %v`, query, id)
	if err != nil {
		log.Printf("Error deleting directory with ID %v: %v", id, err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to delete directory",
		})
	}

	// d := models.IndexedDirs{}
	// err := d.Delete(id)
	// if err != nil {
	// 	if err.Error() == fmt.Sprintf("no directory found with ID %d", id) {
	// 		return c.JSON(http.StatusNotFound, map[string]string{
	// 			"error": "Directory not found",
	// 		})
	// 	}

	// 	log.Printf("Error deleting directory with ID %v: %v", id, err)
	// 	return c.JSON(http.StatusInternalServerError, map[string]string{
	// 		"error": "Failed to delete directory",
	// 	})
	// }

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Directory deleted successfully",
	})
	// Return success response
}

func explainAnalyze(stmt string, placeholders ...interface{}) {
	conn, err := utils.PgConn()
	if err != nil {
		log.Println("Error connecting to the database for search words:", err)
		return
	}
	defer conn.Close()
	explainAnalyze := fmt.Sprintf("EXPLAIN ANALYZE %s", stmt)
	explainRows, err := conn.Query(explainAnalyze, placeholders...)
	if err != nil {
		log.Println("Error running EXPLAIN ANALYZE:", err)
	}
	defer explainRows.Close()
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
}

func insertSearchedWord(word string) {
	const addSearchWords = `INSERT INTO search (input) VALUES ($1);`
	conn, err := utils.PgConn()
	if err != nil {
		log.Println("Error connecting to the database for search words:", err)
		return
	}
	defer conn.Close()
	log.Println("Inserting search words into the database:", word)
	_, err = conn.Exec(addSearchWords, word)
	if err != nil {
		log.Println("Error inserting search words:", err)
	}
}

func SelectFinfoById(id string) (models.Finfo, error) {
	f := models.Finfo{}
	stmt := `SELECT id, name, ext, is_dir, directory, size, mod_time FROM files WHERE id = $1;`

	conn, err := utils.PgConn()
	if err != nil {
		return f, err
	}
	defer conn.Close()

	err = conn.QueryRow(stmt, id).Scan(
		&f.Id,
		&f.Name,
		&f.Ext,
		&f.IsDir,
		&f.Directory,
		&f.Size,
		&f.ModTime,
	)

	if err != nil {
		return f, fmt.Errorf("failed to select file info by ID: %w", err)
	}

	_ = f.CheckExtension()

	return f, nil
}
