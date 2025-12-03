package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func main() {

	path := os.Args[1]

	fetched, err := fetchBody(path)
	if err != nil {
		panic(err)
	}

	document, err := html.Parse(strings.NewReader(fetched))
	fmt.Println(document)
	if err != nil {
		panic(err)
	}
	for _, node := range visit(nil, document) {
		fmt.Println(node)
	}

}

func fetchBody(path string) (string, error) {

	var b []byte
	resp, err := http.Get(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		return "", err
	}
	b, err = io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: odczytywanie %s: %v\n", path, err)
		return "", err
	}

	return string(b), nil
}

func visit(links []string, n *html.Node) []string {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				links = append(links, a.Val)
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		links = visit(links, c)
	}
	return links
}

func printWithParams() {
	firstArg, _ := strconv.Atoi(os.Args[1])
	second, _ := strconv.Atoi(os.Args[2])
	if len(os.Args) < 4 {
		result := add(firstArg, second)
		println("Result:", result)
		return
	}
	third := os.Args[3]

	result := add(firstArg, second, third)

	println("Result:", result)
}

func add(a, b int, params ...string) int {
	if len(params) > 0 {

		switch {
		case slices.Contains(params, "double"):
			return 2 * (a + b)
		case slices.Contains(params, "triple"):
			return 3 * (a + b)
		case slices.Contains(params, "quadruple"):
			return 4 * (a + b)
		default:
			return a + b
		}
	}
	return a + b
} // do something with params
