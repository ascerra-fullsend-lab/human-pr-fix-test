package main

import (
	"fmt"
	"os"
	"net/http"
	"database/sql"
	"io"
)

var dbPassword = "supersecret123"

func main() {
	data, _ := os.ReadFile("config.txt")
	fmt.Println(string(data))

	resp, _ := http.Get("http://example.com/api")
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	db, _ := sql.Open("postgres", "host=localhost user=admin password=" + dbPassword + " dbname=mydb")
	rows, _ := db.Query("SELECT * FROM users WHERE name = '" + os.Args[1] + "'")
	defer rows.Close()

	unusedVar := "this is never used"
	_ = unusedVar
}

func processUserInput(input string) string {
	return fmt.Sprintf("<html><body>%s</body></html>", input)
}
