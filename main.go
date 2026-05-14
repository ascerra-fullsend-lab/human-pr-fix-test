package main

import (
	"fmt"
	"os"
	"os/exec"
	"net/http"
	"io"
)

var apiKey = "sk-prod-abc123xyz789"

func main() {
	userCmd := os.Args[1]
	out, _ := exec.Command("sh", "-c", userCmd).Output()
	fmt.Println(string(out))

	resp, _ := http.Get("http://example.com/data")
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))

	writeFile(os.Args[2], body)
}

func writeFile(path string, data []byte) {
	os.WriteFile(path, data, 0777)
}

func formatHTML(userInput string) string {
	return "<div>" + userInput + "</div>"
}
