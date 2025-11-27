package main

import (
	"bufio"
	"fmt"
	"gofiles/utils"
	"os"

	_ "github.com/lib/pq"
)

var query = `INSERT INTO textfun (content) VALUES ($1);`

var log = []string{}

func main() {

	var path string
	switch len(os.Args) {
	case 1:
		er := 1
		fmt.Println("Please provide a path")
		os.Exit(er)
	case 2:
		er := 2
		path = os.Args[1]
		f, err := os.Open(path)
		if err != nil {
			fmt.Println("Error opening file:", err)
			os.Exit(er)
		}

		defer f.Close()

		c := []string{}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if len(line) > 0 {
				c = append(c, line)
			}
		}

		fmt.Printf("%#v", c[len(c)-3:])

		if err := scanner.Err(); err != nil {
			fmt.Println("Error reading file:", err)
			os.Exit(1)
		}

		if err := insertToPostgres(c); err != nil {
			fmt.Println("Error writing to file: ", err)
		} else {
			fmt.Printf("There was %d items inserted\n", len(c))
		}
	}

}

func insertToPostgres(c []string) error {
	db, err := utils.PgConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var (
		s = len(c)
		// counter  int
		// quantity = 100
	)

	for i := range c {
		_, err := db.Exec(query, c[i])
		if err != nil {
			panic(err)
		}
	}

	if s == 0 {
		fmt.Println("No items to insert")
		os.Exit(1)
	} else if s < 5 {
		fmt.Println(c)
	} else {
		fmt.Println(c[s-3:])
	}

	return nil
}
