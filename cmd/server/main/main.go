package main

import (
	"crypto/subtle"
	"gofiles/internal/utils"
	"html/template"
	"io"

	"gofiles/internal/handlers"

	_ "net/http/pprof"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	// Middleware
	// e.Use(middleware.Logger())
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())

	protected := e.Group("", middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
		// Be careful to use constant time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(username), []byte("admin")) == 1 &&
			subtle.ConstantTimeCompare([]byte(password), []byte("secret")) == 1 {
			return true, nil
		}
		e.GET("/", handlers.StartPage)
		return false, nil
	}))

	e.Static("/static", "static")
	e.Static("/media", "media")

	e.Renderer = &Template{
		templates: template.Must(template.New("").Funcs(template.FuncMap{
			"formatDate": utils.FormatDate, // Register the custom function
			"not":        utils.Not,
			"equals":     utils.Equals,
			"notequals":  utils.Notequals,
		}).ParseGlob("public/views/*.html")),
	}

	e.GET("/", handlers.StartPage)
	e.GET("/search", handlers.SearchInDb)
	e.POST("/search", handlers.SearchInDb)
	e.GET("/detail/:id", handlers.DetailById)
	e.GET("/details/:id", handlers.DetailById)
	e.GET("/preview/:id", handlers.PreviewById)
	e.GET("/preview/image/:id", handlers.PreviewImage)
	e.GET("/json/search", handlers.ResponseJson)
	e.GET("/append", handlers.ResponseAppend)
	e.GET("/access", Accessible)

	protected.GET("/dirs", handlers.AddPath)
	protected.POST("/dirs", handlers.AddPath)
	protected.POST("/scan", handlers.ScanDirectory)
	protected.DELETE("/dirs/delete/:id", handlers.DeleteDirectory)
	protected.GET("/test", handlers.TestEndpoint)

	e.Logger.Fatal(e.Start(":80"))

}
