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

func AddPath(c echo.Context) error {

	i := models.IndexedDirs{Params: make(map[string]string), Error: make(map[string]string)}
	method := c.Request().Method
	i.HeaderTitle = "Add Path To Index"

	switch method {

	case http.MethodPost:
		params := c.FormValue("path")
		i.Params["path"] = params
		_ = i.SetParams(c)

		i.Status = true

		if i.Status == true {
			_ = i.Append()
		}

		if err := i.List(); err != nil {
			return c.String(http.StatusInternalServerError, "Error listing indexed directories: "+err.Error())
		}

		return c.Redirect(http.StatusSeeOther, "/dirs")

		// if err := c.Render(http.StatusOK, "dirs", i); err != nil {
		// 	return c.String(http.StatusInternalServerError, "Error rendering add page: "+err.Error())
		// }

	case http.MethodGet:
		if err := i.List(); err != nil {
			return c.String(http.StatusInternalServerError, "Error listing indexed directories: "+err.Error())
		}

		err := c.Render(http.StatusOK, "dirs", i)
		if err != nil {
			return c.String(http.StatusInternalServerError, "Error rendering add page: "+err.Error())
		}
		return nil
	}

	return nil
}
