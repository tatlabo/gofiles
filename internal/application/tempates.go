package application

import (
	"bytes"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"gofiles/ui"
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

type Utils struct {
	FormatDate func(time.Time) string
	Not        func(bool) bool
	Equals     func(a, b any) bool
	Notequals  func(a, b any) bool
}

var utils = Utils{
	FormatDate: formatDate,
	Not:        not,
	Equals:     equals,
	Notequals:  notequals,
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05") // Customize the format as needed
}

func not(b bool) bool {
	return !b
}

func equals(a, b any) bool {
	return a == b
}

func notequals(a, b any) bool {
	return a != b
}
