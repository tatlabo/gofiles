package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "path_validation_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a temporary file for testing
	tempFile, err := os.CreateTemp(tempDir, "testfile.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid directory path",
			path:        tempDir,
			expectError: false,
		},
		{
			name:        "Empty path",
			path:        "",
			expectError: true,
			errorMsg:    "path cannot be empty",
		},
		{
			name:        "Path traversal attempt",
			path:        "../../../etc/passwd",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "Non-existent path",
			path:        "/this/path/does/not/exist/at/all",
			expectError: true,
			errorMsg:    "path does not exist",
		},
		{
			name:        "Path is a file, not directory",
			path:        tempFile.Name(),
			expectError: true,
			errorMsg:    "path is not a directory",
		},
		{
			name:        "Whitespace around valid path",
			path:        "  " + tempDir + "  ",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePath(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', but got no error", tt.errorMsg)
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', but got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
				if result == "" {
					t.Error("Expected non-empty result for valid path")
				}
				// Check that the result is an absolute path
				if !filepath.IsAbs(result) {
					t.Errorf("Expected absolute path, got: %s", result)
				}
			}
		})
	}
}

func TestValidatePath_PathTraversalVariations(t *testing.T) {
	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"./../../sensitive",
		"path/../../../etc",
		"/some/path/../../etc/passwd",
	}

	for _, path := range pathTraversalAttempts {
		t.Run(path, func(t *testing.T) {
			_, err := ValidatePath(path)
			if err == nil {
				t.Errorf("Expected error for path traversal attempt: %s", path)
			}
			if !containsString(err.Error(), "path traversal detected") {
				t.Errorf("Expected 'path traversal detected' error, got: %v", err)
			}
		})
	}
}

func TestValidatePath_ReturnsAbsolutePath(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "abs_path_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test with relative path (if we can construct one)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	// Change to temp directory temporarily
	err = os.Chdir(tempDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(cwd)

	// Create a subdirectory in tempDir
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Test with just the subdirectory name (relative path)
	result, err := ValidatePath("subdir")
	if err != nil {
		t.Errorf("Expected no error for valid relative path, got: %v", err)
	}

	if !filepath.IsAbs(result) {
		t.Errorf("Expected absolute path, got relative: %s", result)
	}

	// Verify the absolute path is correct
	expectedAbs, _ := filepath.Abs("subdir")
	if result != expectedAbs {
		t.Errorf("Expected absolute path %s, got %s", expectedAbs, result)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
