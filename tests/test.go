package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"
	"unicode"

	"golang.org/x/net/html"

	"sync"
)

type FinfoJSON struct {
	Directory string    `json:"directory"`
	Name      string    `json:"name"`
	Ext       string    `json:"ext"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modTime"`
}

type person struct{}
type human struct{}
type animal struct{}

func (p person) hello() {
	fmt.Println("hello Person")
}

func (a animal) hello() {
	fmt.Println("hello Animal")
}

// error: hello redeclared in this block
func (h human) hello() {
	fmt.Println("hello Human")
}

type mammal interface {
	hello()
}

func hello(m mammal) {
	m.hello()
}

func IsPalindrome(s string) bool {
	word := []rune{}
	for _, r := range s {
		if unicode.IsLetter(r) {
			word = append(word, unicode.ToLower(r))
		}
	}

	for i := range word {
		if word[i] != word[len(word)-1-i] {
			return false
		}
	}
	return true
}

func main() {

	// path := os.Args[1]

	// fetched, err := fetchBody(path)
	// if err != nil {
	// 	panic(err)
	// }

	// document, err := html.Parse(strings.NewReader(fetched))
	// fmt.Println(document)
	// if err != nil {
	// 	panic(err)
	// }
	// for _, node := range visit(nil, document) {
	// 	fmt.Println(node)
	// }

	j, _ := json.Marshal(FinfoJSON{
		Directory: "C:/test",
		Name:      "file.txt",
		Ext:       ".txt",
		IsDir:     false,
		Size:      1234,
		ModTime:   time.Now(),
	})

	fmt.Println(string(j))

	p := person{}
	h := human{}
	a := animal{}

	hello(h)

	hello(p)

	hello(a)

	h.hello()

	r := new(result)
	r.Word = "example"
	r.Count = 42
	r.Data = map[string]any{
		"key1": "value1",
		"key2": 123,
	}

	http.Handle("/count", new(countHandler))
	http.Handle("/result", r)
	log.Fatal(http.ListenAndServe(":8080", nil))

}

type countHandler struct {
	mu sync.Mutex // guards n
	n  int
}

func (h *countHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.n++
	fmt.Fprintf(w, "count is %d\n", h.n)
}

type result struct {
	Word  string
	Count int
	Data  map[string]any
}

func (r *result) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "Word: %s\nCount: %d\nData: %v\n", r.Word, r.Count, r.Data)
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
