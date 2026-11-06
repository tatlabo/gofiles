// Package static; serve js, css, images
package static

import "embed"

//go:embed "js" "css" "images"
var StaticFiles embed.FS
