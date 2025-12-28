package main

import (
	"fmt"
	"gofiles/internal/models"
	"log"
	"os"
	"strings"
)

func main() {

	var path string

	switch len(os.Args) {
	case 1:
		fmt.Println("Please provide a path")
		os.Exit(1)
	case 2:
		path = os.Args[1]
		path = strings.TrimSpace(path)
		path = strings.ReplaceAll(path, "/", "\\")
		_, err := os.ReadDir(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
			os.Exit(1)
		}

	}

	d := models.Directory{}
	err := d.AddPath(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error adding path:")
		fmt.Fprintf(os.Stderr, "%v\n", err)
		log.Fatal(err)
	}

	fmt.Println("Path added:")
	fmt.Printf("Id: %v\n", d.Id)
	fmt.Printf("Path: %v\n", d.Path)
	fmt.Printf("IsDone: %v\n", d.IsDone)
	fmt.Printf("CreatedAt: %v\n", d.CreatedAt)
	fmt.Printf("UpdatedAt: %v\n", d.UpdatedAt)
	// err := scan.Scan(d)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
	// }

	// err := insertIntoDirs(path)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error inserting into directory table: %v\n", err)
	// 	return err
	// }
}
