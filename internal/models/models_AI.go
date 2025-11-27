// Package models contains data structures and methods for handling file information
package models

import (
	"fmt"
	"gofiles/utils"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

var text = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv", "R", "r"}
var image = []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "svg", "ico", "heic", "raw"}
var video = []string{"mp4", "wav", "mp3", "aif", "aiff", "avi", "mov", "wmv", "flv", "webm"}
var archive = []string{"zip", "rar", "7z", "tar", "gz", "bz2", "xz", "tar.gz", "tar.bz2"}
var executable = []string{"exe", "msi", "deb", "rpm", "dmg", "app", "com", "scr"}

type Item struct {
	Id           int          `db:"id" json:"id"`
	ParentId     int          `db:"parent_id" json:"parentId"`
	Path         string       `db:"path" json:"path"`
	Name         string       `db:"name" json:"name"`
	Ext          string       `db:"ext" json:"ext"`
	IsDir        bool         `db:"is_dir" json:"isDir"`
	Size         int64        `db:"size" json:"size"`
	SizeStr      string       `json:"sizeStr"`
	ModTime      time.Time    `db:"mod_time" json:"modTime"`
	CreatedAt    time.Time    `db:"created_at" json:"createdAt"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updatedAt"`
	IsImage      bool         `json:"isImage"`
	IsText       bool         `json:"isText"`
	IsVideo      bool         `json:"isVideo"`
	IsArchive    bool         `json:"isArchive"`
	IsExecutable bool         `json:"isExecutable"`
	Link         string       `json:"link"`
	Src          template.URL `json:"src"`
	TsRank       float64      `db:"ts_rank" json:"tsRank"`    // For full-text search ranking
	Checksum     string       `db:"checksum" json:"checksum"` // MD5/SHA256 hash for file integrity
	MimeType     string       `db:"mime_type" json:"mimeType"`
	Keywords     string       `db:"keywords" json:"keywords"`       // For full-text search
	Tags         string       `db:"tags" json:"tags"`               // User-defined tags
	IsHidden     bool         `db:"is_hidden" json:"isHidden"`      // Hidden files/folders
	IsSymlink    bool         `db:"is_symlink" json:"isSymlink"`    // Symbolic links
	LinkTarget   string       `db:"link_target" json:"linkTarget"`  // Target of symlink
	Permissions  string       `db:"permissions" json:"permissions"` // File permissions (e.g., "rwxr-xr-x")
	Owner        string       `db:"owner" json:"owner"`             // File owner
	Group        string       `db:"group" json:"group"`             // File group
	Depth        int          `db:"depth" json:"depth"`             // Directory depth level
	FileCount    int          `db:"file_count" json:"fileCount"`    // Number of files in directory (if IsDir)
	TotalSize    int64        `db:"total_size" json:"totalSize"`    // Total size including subdirectories
}

type ItemDetail struct {
	*Item
	Title   string
	Preview string
	HTML    template.HTML
}

func (f *Item) CheckExtension() error {

	if f.IsDir {
		fSrc := strings.ReplaceAll(f.Path+"\\"+f.Name, "\\", "/")
		f.Src = template.URL(fSrc)
		return nil
	}

	if slices.Contains(textFiles, f.Ext) {
		f.IsText = true
		f.Link = fmt.Sprintf("%s\\%s.%v", f.Path, f.Name, f.Ext)

	} else if slices.Contains(imageFiles, f.Ext) {
		f.IsImage = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Path, f.Name, f.Ext)
	} else if slices.Contains(videoFiles, f.Ext) {
		f.IsVideo = true
		f.Link = fmt.Sprintf("file:///%s\\%s.%v", f.Path, f.Name, f.Ext)
	}

	f.Link = strings.ReplaceAll(f.Link, "\\", "/")
	fSrc := fmt.Sprintf("%s\\%s.%v", f.Path, f.Name, f.Ext)
	fSrc = strings.ReplaceAll(fSrc, "\\", "/")
	f.Src = template.URL(fSrc)

	return nil
}

func (f *Item) String() string {
	return fmt.Sprintf("%s, %s, %s, %T\n", f.Path, f.Name, f.Ext, f.IsDir)
}

type IndexPage struct {
	TC          []Finfo `json:"FileList"`
	Text        string
	HeaderTitle string
	Counter     int
	Params      map[string]string
	Error       map[string]string
}

type SearchParameters struct {
	Params       string
	Like         string
	Dir          string
	Keywords     string
	Limit        int
	Offset       int
	QueryParam   string
	Stmt         string
	CounterStmt  string
	Ext          string
	Placeholders []any
	Error        map[string]string
}

func (s *SearchParameters) QueryStmt() error {

	switcher := 0
	isOn := []string{"on", "true", "1", "yes", "ok", "y", "tak", "t"}

	if slices.Contains(isOn, s.Like) {
		switcher += 1
	}

	if slices.Contains(isOn, s.Dir) {
		switcher += 100
	}

	language := "'polish'"
	if len(s.Keywords) > 0 {
		language = "'english'"
	}

	s.QueryParam = s.Params

	if strings.Contains(s.Params, ".") {
		// Regex: split on dot, comma, semicolon, or any whitespace
		re := regexp.MustCompile(`[.,;]+`)
		parts := re.Split(s.Params, -1)
		s.QueryParam = parts[0]
		s.QueryParam = strings.TrimSpace(s.QueryParam)
		s.Ext = parts[1]
	}

	switch switcher {
	case 0:

		// column = keywords
		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, path, size, mod_time,
		ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
		FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
		ORDER BY ts_rank DESC
		LIMIT $2 OFFSET $3;`, language)

		s.CounterStmt = fmt.Sprintf(`
		SELECT COUNT(id) FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, language)

	case 1:
		s.QueryParam = "^" + s.QueryParam
		// column = "name"
		s.Stmt = `
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1)
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`

		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1);`
	case 100:

		// column = "files.keywords"

		s.Stmt = fmt.Sprintf(`
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`, language)

		s.CounterStmt = fmt.Sprintf(`
		SELECT COUNT(*) FROM files 
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND is_dir=true;`, language)

	case 101:
		s.QueryParam = "^" + s.QueryParam

		s.Stmt = `
		SELECT id, name, ext, is_dir, path, size, mod_time FROM files
		WHERE LOWER(name) ~ LOWER($1) AND is_dir=true
		ORDER BY mod_time DESC LIMIT $2 OFFSET $3;`
		s.CounterStmt = `SELECT COUNT(*) FROM files WHERE LOWER(name) ~ LOWER($1) AND is_dir=true;`
	}

	s.Placeholders = []any{s.QueryParam, s.Limit, s.Offset}

	if len(s.Ext) > 0 {

		s.Stmt = fmt.Sprintf(`SELECT files.id, name, files.ext, is_dir, path, size, mod_time,
		ts_rank( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
		FROM files
		JOIN ext ON files.ext_id = ext.id
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND ext.ext = $2
		ORDER BY ts_rank DESC LIMIT $3 OFFSET $4;`, language)

		s.CounterStmt = fmt.Sprintf(`SELECT COUNT(*) FROM files
		JOIN ext ON files.ext_id = ext.id
		WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, files.keywords)
		AND ext.ext = $2;`, language)
		s.Placeholders = []any{s.Placeholders[0], s.Ext, s.Placeholders[1], s.Placeholders[2]}
	}

	// s.ExplainAnalyze = fmt.Sprintf(`EXPLAIN ANALYZE %s`, s.Stmt)

	return nil

}

func (sp *SearchParameters) SetParams(c echo.Context) error {

	method := c.Request().Method

	switch method {

	case http.MethodGet:

		if len(c.QueryParam("name")) == 0 {
			sp.Error = map[string]string{"Error": "No search parameters provided"}
		} else {
			sp.Params = utils.CleanInput(c.QueryParam("name"))

			if len(c.QueryParam("like")) > 0 {
				sp.Like = utils.CleanInput(c.QueryParam("like"))
			}

			if len(c.QueryParam("dir")) > 0 {
				sp.Dir = utils.CleanInput(c.QueryParam("dir"))
			}

			if len(c.QueryParam("keywords")) > 0 {
				sp.Keywords = utils.CleanInput(c.QueryParam("keywords"))
			}

			offsetStr := c.QueryParam("offset")
			limitStr := c.QueryParam("limit")

			if len(limitStr) > 0 {
				sp.Limit, _ = strconv.Atoi(limitStr)
			} else {
				sp.Limit = 10
			}

			if len(offsetStr) > 0 {
				sp.Offset, _ = strconv.Atoi(offsetStr)
			} else {
				sp.Offset = 0
			}
		}

	case http.MethodPost:
		params := c.FormValue("name")
		sp.Params = utils.CleanInput(params)

		if len(c.FormValue("like")) > 0 {
			sp.Like = c.FormValue("like")
		}

		if len(c.FormValue("dir")) > 0 {
			sp.Dir = c.FormValue("dir")
		}

		if len(c.FormValue("keywords")) > 0 {
			sp.Keywords = c.FormValue("keywords")
		}

		sp.Limit = 10
		sp.Offset = 0

	}

	return nil
}

func (f *Item) BeforeCreate() {
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
}

// BeforeUpdate sets the updated timestamp
func (f *Item) BeforeUpdate() {
	f.UpdatedAt = time.Now()
}

// GetFullPath returns the complete file path
func (f *Item) GetFullPath() string {
	if f.IsDir {
		return filepath.Join(f.Path, f.Name)
	}
	if f.Ext != "" {
		return filepath.Join(f.Path, f.Name+"."+f.Ext)
	}
	return filepath.Join(f.Path, f.Name)
}

// GetHumanReadableSize returns a human-readable size string
func (f *Item) GetHumanReadableSize() string {
	if f.SizeStr != "" {
		return f.SizeStr
	}
	return formatFileSize(f.Size)
}

// formatFileSize converts bytes to human readable format
func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsModifiedSince checks if file was modified after given time
func (f *Item) IsModifiedSince(t time.Time) bool {
	return f.ModTime.After(t)
}

// SetFileType determines file type based on extension
func (f *Item) SetFileType() {
	if f.IsDir {
		return
	}

	ext := strings.ToLower(f.Ext)

	// Reset all type flags
	f.IsImage = false
	f.IsText = false
	f.IsVideo = false
	f.IsArchive = false
	f.IsExecutable = false

	// Check file types
	if slices.Contains(text, ext) {
		f.IsText = true
	} else if slices.Contains(image, ext) {
		f.IsImage = true
	} else if slices.Contains(video, ext) {
		f.IsVideo = true
	} else if slices.Contains(archive, ext) {
		f.IsArchive = true
	} else if slices.Contains(executable, ext) {
		f.IsExecutable = true
	}
}

// GenerateChecksum calculates file checksum (implement based on your needs)
func (f *Item) GenerateChecksum() error {
	if f.IsDir {
		return nil
	}

	// This is a placeholder - you would implement actual checksum calculation
	// For now, we'll just set it to empty or use a simple hash
	f.Checksum = ""
	return nil
}

// SetMimeType determines MIME type based on extension
func (f *Item) SetMimeType() {
	if f.IsDir {
		f.MimeType = "inode/directory"
		return
	}

	ext := strings.ToLower(f.Ext)
	mimeTypes := map[string]string{
		"txt":  "text/plain",
		"html": "text/html",
		"css":  "text/css",
		"js":   "application/javascript",
		"json": "application/json",
		"pdf":  "application/pdf",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"mp4":  "video/mp4",
		"mp3":  "audio/mpeg",
		"zip":  "application/zip",
		"go":   "text/x-go",
		"py":   "text/x-python",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		f.MimeType = mimeType
	} else {
		f.MimeType = "application/octet-stream"
	}
}

// Validate performs basic validation on the struct
func (f *Item) Validate() error {
	if f.Path == "" {
		return fmt.Errorf("path cannot be empty")
	}
	if f.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if !f.IsDir && f.Size < 0 {
		return fmt.Errorf("file size cannot be negative")
	}
	return nil
}

// TableName returns the database table name
func (f *Item) TableName() string {
	return "files"
}
