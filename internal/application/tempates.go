package application

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"

	"gofiles/ui"
	"gofiles/utils"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w http.ResponseWriter, templateName string, data any) {
	tmpltBuf := &bytes.Buffer{}
	err := t.templates.ExecuteTemplate(tmpltBuf, templateName, data)
	if err != nil {
		msg := fmt.Sprintf("template execting failed, %v", err)
		_, _ = w.Write([]byte(msg))
		slog.Error("tempaltes failed", "start", err)

	}

	pos, err := tmpltBuf.WriteTo(w)
	if err != nil {
		msg := fmt.Sprintf("template execting failed, %v", err)
		t.HTMLError(w, msg)
		slog.Error("tempaltes failed", "temlateName", err, "position", pos)
	}
}

// func (app *Application) Render(t *Template, w http.ResponseWriter, templateName string, data any) {
// 	tmpltBuf := &bytes.Buffer{}
// 	err := t.templates.ExecuteTemplate(tmpltBuf, templateName, data)
// 	if err != nil {
// 		msg := fmt.Sprintf("template execting failed, %v", err)
// 		_, _ = w.Write([]byte(msg))
// 		slog.Error("tempaltes failed", "start", err)

// 	}

// 	pos, err := tmpltBuf.WriteTo(w)
// 	if err != nil {
// 		msg := fmt.Sprintf("template execting failed, %v", err)
// 		t.HTMLError(w, msg)
// 		slog.Error("tempaltes failed", "temlateName", err, "position", pos)
// 	}
// }

var tmpl = Template{
	templates: template.Must(template.New("").Option("missingkey=zero").Funcs(funcMap()).ParseFS(ui.HTML, "html/*html")),
}

var funcMap = func() template.FuncMap {
	return template.FuncMap{
		"formatDate": utils.FormatDate,
		"not":        utils.Not,
		"equals":     utils.Equals,
		"notequals":  utils.Notequals,
	}
}

func (t *Template) HTMLError(w http.ResponseWriter, msg string) {
	_, _ = w.Write([]byte(msg))
}
