package chroma

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

func HighlightCode(filePath string) (string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}

	// Get the lexer for the specified language
	lexer := lexers.Match(filePath)
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// If a specific language is provided, use it
	lexer = chroma.Coalesce(lexer)

	// Use the Monokai style (similar to VS Code's default dark theme)
	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	// Create an HTML formatter with line numbers
	formatter := html.New(html.WithLineNumbers(false), html.WithClasses(true))

	// Tokenize the content
	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return "", fmt.Errorf("error tokenizing content: %v", err)
	}

	// Generate the highlighted HTML
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", fmt.Errorf("error formatting content: %v", err)
	}

	return buf.String(), nil
}
