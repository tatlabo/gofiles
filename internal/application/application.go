package application

import (
	"html/template"
	"log/slog"

	"gofiles/internal/models"
	"gofiles/ui"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Application struct {
	Log *slog.Logger
	DB  *pgxpool.Pool

	BasicAuthUsers UserApp
	Template

	File models.Filer
	Dir  models.Direr
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
		Log:      logger,
		DB:       db,
		File:     &models.FileModel{DB: db},
		Dir:      &models.DirectoryModel{DB: db},
	}

	app.Log.Debug("application", "Run", "OK")
	return app
}
