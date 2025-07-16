package main

import (
	"gofiles/internal/utils"
	"html/template"
	"io"
	"log"
	"net/http"

	"gofiles/internal/handlers"

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

func main() {

	// Initialize Echo framework
	e := echo.New()

	t := &Template{
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"formatDate": utils.FormatDate, // Register the custom function
			"not":        utils.Not,
			"equals":     utils.Equals,
			"notequals":  utils.Notequals,
		}).ParseGlob("public/views/*.html")),
	}

	e.Static("/static", "static")
	e.Static("/media", "media")
	e.Renderer = t

	e.GET("/", handlers.StartPage)
	e.GET("/search", handlers.SearchInDb) // FindForm
	e.POST("/search", handlers.SearchInDb)
	e.GET("/detail/:id", handlers.DetailById)
	e.GET("/details/:id", handlers.DetailById)
	e.GET("/preview/:id", handlers.PreviewById)
	e.GET("/preview/image/:id", handlers.PreviewImage)
	e.GET("/add", handlers.AddPath)
	e.POST("/add", handlers.AddPath)
	e.DELETE("/delete/:id", handlers.DeleteIndexedDirectory) // New delete route

	// e.GET("/files", findInDb)

	e.GET("/json/search", handlers.ResponseJson)

	e.GET("/append", handlers.ResponseAppend)

	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	e.Logger.Fatal(e.Start(":80"))

}
