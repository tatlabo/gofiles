package application

import (
	"html/template"
	"log/slog"

	"gofiles/ui"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Application struct {
	Logger *slog.Logger
	DB     *pgxpool.Pool

	BasicAuthUsers UserApp
	Template
}

type UserApp struct {
	User string
	Pass string
}

func Run(db *pgxpool.Pool, logger *slog.Logger) (app *Application) {
	//
	// initize tempaltes
	tmpl := Template{
		templates: template.Must(template.New("").Option("missingkey=zero").Funcs(funcMap()).ParseFS(ui.HTML, "html/*html")),
	}

	app = &Application{
		Template: tmpl,
		Logger:   logger,
		DB:       db,
	}

	app.Logger.Debug("application", "Run", "OK")
	return app
}
