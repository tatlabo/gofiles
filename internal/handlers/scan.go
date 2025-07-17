package handlers

import (
	"gofiles/scan"
	"net/http"

	"github.com/labstack/echo/v4"
)

// TestEndpoint is a simple test endpoint
func TestEndpoint(c echo.Context) error {
	return c.String(http.StatusOK, "Test endpoint working!")
}

func ScanDirectory(c echo.Context) error {
	path := c.FormValue("path")
	if path == "" {
		return c.String(http.StatusBadRequest, "Path is required")
	}

	// Scan the directory
	if err := scan.ScanDir(path); err != nil {
		return c.String(http.StatusInternalServerError, "Failed to scan directory")
	}

	return c.String(http.StatusOK, "Directory scanned successfully")
}
