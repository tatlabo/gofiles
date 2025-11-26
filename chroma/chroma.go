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
	style := styles.Get("manni")
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

// GenerateCSS generates CSS for a specific Chroma style
func GenerateCSS(styleName string) (string, error) {
	// Get the style
	style := styles.Get(styleName)
	if style == nil {
		return "", fmt.Errorf("style '%s' not found", styleName)
	}

	// Create HTML formatter with classes
	formatter := html.New(
		html.WithClasses(true),
		html.WithLineNumbers(true),
	)

	// Generate CSS
	var buf bytes.Buffer
	err := formatter.WriteCSS(&buf, style)
	if err != nil {
		return "", fmt.Errorf("error generating CSS: %v", err)
	}

	return buf.String(), nil
}

// SaveCSSToFile saves generated CSS to a file
func SaveCSSToFile(styleName, outputPath string) error {
	css, err := GenerateCSS(styleName)
	if err != nil {
		return err
	}

	err = os.WriteFile(outputPath, []byte(css), 0644)
	if err != nil {
		return fmt.Errorf("error writing CSS file: %v", err)
	}

	return nil
}

// GetAvailableStyles returns list of all available Chroma styles
func GetAvailableStyles() []string {
	return styles.Names()
}

// GenerateAllStylesCSS generates CSS files for all available styles
func GenerateAllStylesCSS(outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	styleNames := styles.Names()
	successCount := 0

	for _, styleName := range styleNames {
		css, err := GenerateCSS(styleName)
		if err != nil {
			fmt.Printf("Warning: failed to generate %s: %v\n", styleName, err)
			continue
		}

		outputPath := fmt.Sprintf("%s/chroma-%s.css", outputDir, styleName)
		err = os.WriteFile(outputPath, []byte(css), 0644)
		if err != nil {
			fmt.Printf("Warning: failed to write %s: %v\n", styleName, err)
			continue
		}

		fmt.Printf("✓ Generated: %s\n", outputPath)
		successCount++
	}

	fmt.Printf("\nGenerated %d/%d styles successfully\n", successCount, len(styleNames))
	return nil
}

// HighlightCodeWithStyle generates highlighted HTML with a specific style
func HighlightCodeWithStyle(filePath string, styleName string) (htmlOutput string, cssOutput string, err error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", "", fmt.Errorf("error reading file: %v", err)
	}

	// Get the lexer
	lexer := lexers.Match(filePath)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Get the style
	style := styles.Get(styleName)
	if style == nil {
		style = styles.Fallback
	}

	// Create formatter with classes
	formatter := html.New(
		html.WithClasses(true),
		html.WithLineNumbers(true),
		html.LineNumbersInTable(true),
		html.TabWidth(4),
	)

	// Tokenize
	iterator, err := lexer.Tokenise(nil, string(content))
	if err != nil {
		return "", "", fmt.Errorf("error tokenizing content: %v", err)
	}

	// Generate HTML
	var htmlBuf bytes.Buffer
	err = formatter.Format(&htmlBuf, style, iterator)
	if err != nil {
		return "", "", fmt.Errorf("error formatting content: %v", err)
	}

	// Generate CSS
	var cssBuf bytes.Buffer
	err = formatter.WriteCSS(&cssBuf, style)
	if err != nil {
		return "", "", fmt.Errorf("error generating CSS: %v", err)
	}

	return htmlBuf.String(), cssBuf.String(), nil
}
