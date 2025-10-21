package main

import (
	"fmt"
	_ "fmt"
	"io/ioutil"
	_ "io/ioutil"
	"net/http"
	_ "net/http"
	"os"
	_ "os"
)

func main() {
	for _, url := range os.Args[1:] {
		resp, error := http.Get(url)
		if error != nil {
			fmt.Println("Error fetching URL:", url, "Error:", error)
			os.Exit(1)
		}
	}
	b, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
}
