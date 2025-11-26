package main

import (
	"flag"
	"fmt"
	"gofiles/chroma"
	"os"
)

func main() {
	// Define flags
	outputDir := flag.String("output", "static", "Output directory for CSS files")
	styleName := flag.String("style", "", "Specific style to generate (empty = all)")
	list := flag.Bool("list", false, "List all available styles")

	flag.Parse()

	// List styles
	if *list {
		styles := chroma.GetAvailableStyles()
		fmt.Println("Available Chroma styles:")
		fmt.Println("========================")
		for i, s := range styles {
			fmt.Printf("%2d. %s\n", i+1, s)
		}
		fmt.Printf("\nTotal: %d styles\n", len(styles))
		return
	}

	// Generate specific style
	if *styleName != "" {
		outputPath := fmt.Sprintf("%s/chroma-%s.css", *outputDir, *styleName)

		fmt.Printf("Generating CSS for style: %s\n", *styleName)
		err := chroma.SaveCSSToFile(*styleName, outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Generated: %s\n", outputPath)
		return
	}

	// Generate all styles
	fmt.Println("Generating CSS for all styles...")
	fmt.Printf("Output directory: %s\n", *outputDir)
	fmt.Println("========================")

	err := chroma.GenerateAllStylesCSS(*outputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✓ Done!")
}
