package handlers

import (
	_ "fmt"
	_ "gofiles/chroma"
	"gofiles/internal/models"
	_ "gofiles/internal/models"
	_ "html/template"
	_ "io"
	_ "log"
	"net/http"
	_ "net/http"
	_ "os"
	_ "strconv"
	_ "time"

	"github.com/labstack/echo/v4"
)

/*

	const indexedDirs = `CREATE TABLE IF NOT EXISTS indexed (
	id SERIAL PRIMARY KEY,
	path TEXT NOT NULL UNIQUE,
	done BOOLEAN NOT NULL DEFAULT FALSE,
	created TIMESTAMPTZ NOT NULL DEFAULT NOW());`

*/

func AddPath(c echo.Context) error {

	i := models.IndexedDirs{Params: make(map[string]string), Error: make(map[string]string)}
	method := c.Request().Method

	switch method {

	case http.MethodPost:
		params := c.FormValue("path")
		i.Params["path"] = "Some string" + params

		i.Status = true

		_ = i.SetParams(c)

		if i.Status == true {
			_ = i.Append()
		}

		if err := i.List(); err != nil {
			return c.String(http.StatusInternalServerError, "Error listing indexed directories: "+err.Error())
		}

		i.Text = "path handler is not implemented yet"
		i.HeaderTitle = "Add Path To Index"

		if err := c.Render(http.StatusOK, "add", i); err != nil {
			return c.String(http.StatusInternalServerError, "Error rendering add page: "+err.Error())
		}

	case http.MethodGet:
		if err := i.List(); err != nil {
			return c.String(http.StatusInternalServerError, "Error listing indexed directories: "+err.Error())
		}

		i.Text = "path handler is not implemented yet"
		i.HeaderTitle = "Add Path To Index"
		err := c.Render(http.StatusOK, "add", i)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error rendering add page: "+err.Error())
		}
		return nil
	}

	return nil
}
