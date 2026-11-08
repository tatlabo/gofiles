// Package application - main handlers
package application

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gofiles/internal/models"
	"gofiles/scan"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type DirsPage struct {
	Title string

	Msg   string
	Err   error
	Dirs  []models.Directory
	Flash string
}

type MsgKey string

func decodeCookie(name string, r *http.Request) (value string, err error) {
	cookie, err := r.Cookie(name)
	if err != nil {
		return "", err
	}

	decoded, err := base64.StdEncoding.DecodeString(cookie.Value)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func encodeCookie(name, value string) (*http.Cookie, error) {
	Cookie := &http.Cookie{
		Path:     "/admin/dirs",
		MaxAge:   60,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	Cookie.Name = name
	str := base64.StdEncoding.EncodeToString([]byte(value))
	Cookie.Value = str

	return Cookie, nil
}

func (app *Application) HandleDirsGet(w http.ResponseWriter, r *http.Request) {
	templatePage := "dirs.html"
	pageData := DirsPage{
		Title: "katalogi",
	}

	// get err or error from context
	var (
		errKey MsgKey = "err"
	)

	// check context for error
	errMsg, ok := r.Context().Value(errKey).(error)
	if ok {
		if errMsg != nil {
			app.Log.Info("context", "errMsg", errMsg)
			pageData.Err = errMsg
		}
	}

	// directory list
	dirs, err := app.Dir.Rows()
	if err != nil {
		pageData.Err = err
		app.ErrGatherDirs(w, pageData)
		return
	}

	pageData.Dirs = dirs

	cookie, err := decodeCookie("message", r)
	if err == nil {
		pageData.Msg = cookie
		app.Log.Info("cokkie", "message", pageData.Msg)
	}

	http.SetCookie(w, &http.Cookie{
		MaxAge: -1,
		Name:   "message",
	})

	tmpl.Render(w, templatePage, pageData)
}

func (app *Application) HandleDirs(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	path := r.FormValue("path")
	path = strings.TrimSpace(path)

	// postgres codes
	// 23505	Unique violation
	// 23502	Not null violation
	// 23503	Foreign key violation
	// 23514	Check violation

	err := app.Dir.AddPath(path)
	if err != nil {
		// Check for unique constraint violation (SQLSTATE 23505)
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.SQLState() == "23505" {
			app.Log.Warn("adding path", "error", err, "path", path)
			err = fmt.Errorf("path already exists: %s", path)
		}
		// redirect to admin with context err
		app.Log.Error("adding path", "error", err)
		app.HandleDirsGet(w, requestWithErr(err, r))
		return
	}

	// Redirect to /admin with cookie
	cookie, err := encodeCookie("message", "dodano folder do skanowania: "+path)
	if err == nil {
		http.SetCookie(w, cookie)
	}
	//
	redirectURL := fmt.Sprintf("/admin/%s?success=true&added_path=%s", r.URL.Path, path)
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func requestWithMsg(msg string, r *http.Request) *http.Request {
	var msgKey MsgKey = "msg"
	ctx := context.WithValue(r.Context(), msgKey, msg)
	return r.WithContext(ctx)
}

func requestWithErr(err error, r *http.Request) *http.Request {
	var msgKey MsgKey = "err"
	ctx := context.WithValue(r.Context(), msgKey, err)
	return r.WithContext(ctx)
}

func (app *Application) HandleDirDelete(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	id := r.FormValue("id")

	uuid, err := uuid.Parse(id)
	if err != nil {
		// fail
		app.Log.Error("pasing UUID", "failed", err)
		app.HandleDirsGet(w, requestWithErr(err, r))
		return
	}

	path, err := app.Dir.DeleteDir(uuid)
	if err != nil {
		// fail
		// redirect to admin with context err
		app.Log.Error("delete directory", " failed", err)
		app.HandleDirsGet(w, requestWithErr(err, r))
		return
	}

	// success
	// redirect with cookie
	cookie, err := encodeCookie("message", "delete directory success: "+path)
	if err == nil {
		http.SetCookie(w, cookie)
	}
	redirectURL := "/admin/dirs"
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

func (app *Application) HandleScan(w http.ResponseWriter, r *http.Request) {
	templatePage := "dirs.html"

	_ = r.ParseForm()
	id := r.FormValue("scanid")

	uuid, err := uuid.Parse(id)
	if err != nil {
		app.Log.Error("invalid", "UUID", err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	dir, err := app.Dir.Row(uuid)
	if err != nil {

		pageData := DirsPage{
			Title: id,
			Err:   fmt.Errorf("cant't find direcotry with ID %s; error: %w", uuid, err),
		}

		dirs, err := app.Dir.Rows()
		if err != nil {
			pageData.Err = err
			app.ErrGatherDirs(w, pageData)
			return
		}

		pageData.Dirs = dirs

		tmpl.Render(w, templatePage, pageData)
		return
	}

	scanErr := make(chan error)
	go func(d models.Directory) {
		err := scan.Scan(dir)
		if err != nil {
			app.Log.Error("error scanning directory", "path", d.Path, "error", err)
			scanErr <- err
		}
	}(dir)

	// TO DO check if path exists

	select {
	case msgErr := <-scanErr:
		app.Log.Error("scan dir", " failed", msgErr)
		app.HandleDirsGet(w, requestWithErr(msgErr, r))
	case <-time.After(2 * time.Second):
		// scan started
		// redirect with cookie
		cookie, err := encodeCookie("message", "scan started")
		if err == nil {
			http.SetCookie(w, cookie)
		}
		redirectURL := "/admin/dirs"
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func (app *Application) ErrGatherDirs(w http.ResponseWriter, pageData DirsPage) {
	templatePage := "dirs.html"
	app.Log.Error("dirs list", "error", pageData.Err)
	tmpl.Render(w, templatePage, pageData)
}
