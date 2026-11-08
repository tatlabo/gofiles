package models

import (
	"fmt"
	"html/template"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	textFiles  = []string{"py", "txt", "js", "jsx", "json", "css", "go", "html", "edl", "xml", "java", "c", "cpp", "h", "php", "sql", "sh", "bat", "pl", "rb", "swift", "ts", "yaml", "yml", "csv", "R", "r"}
	imageFiles = []string{"jpg", "jpeg", "png", "gif", "bmp", "tif", "tiff", "webp", "svg", "ico", "heic", "raw"}
	videoFiles = []string{"mp4", "wav", "mp3", "aif", "aiff", "mov", "avi", "mkv", "flv", "wmv", "webm", "mpg", "mpeg", "3gp"}
)

type FinfoJSON struct {
	Directory string    `json:"directory"`
	Name      string    `json:"name"`
	Ext       string    `json:"ext"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modTime"`
}

type FileData struct {
	FinfoJSON     `json:"finfo"`
	ID            int          `json:"id"`
	DirectoryID   uuid.UUID    `json:"directoryId"`
	Keywords      string       `json:"keywords"`
	SizeSimple    string       `json:"sizeSimple"`
	ModTimeSimple string       `json:"modTimeStr"`
	Type          string       `json:"type"`
	URL           template.URL `json:"url"`
	TSRank        float64      `json:"tsRank"`
}

type FilesDataList struct {
	List  []FileData
	Count int
}

type QueryParams struct {
	Keywords  string
	Limit     int
	Offset    int
	Order     string
	Ascending bool
	Method    string
}

func (f *FileData) SimplifyDetails() {
	f.SizeSimple = ConvertBytes(f.Size)
	f.ModTimeSimple = f.ModTime.Format("2006-01-02 15:04:05")
	f.CheckExtension()
}

func ConvertBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (f *FileData) CheckExtension() {
	var url string
	f.Type = ""

	if slices.Contains(textFiles, strings.ToLower(f.Ext)) {
		f.Type = "txt"
		url = fmt.Sprintf("%s/%s.%v", f.Directory, f.Name, f.Ext)
	} else if slices.Contains(imageFiles, strings.ToLower(f.Ext)) {
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
		f.Type = "image"
	} else if slices.Contains(videoFiles, strings.ToLower(f.Ext)) {
		f.Type = "video"
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
	} else {
		url = fmt.Sprintf("file:///%s/%s.%v", f.Directory, f.Name, f.Ext)
	}

	f.URL = template.URL(strings.ReplaceAll(url, "\\", "/"))
}
