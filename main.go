package main

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> <output-file>\n", os.Args[0])
		os.Exit(1)
	}

	userCmd := os.Args[1]
	parts := strings.Fields(userCmd)
	out, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	resp, err := http.Get("http://example.com/data")
	if err != nil {
		fmt.Fprintf(os.Stderr, "http error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(body))

	if err := writeFile(os.Args[2], body); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
}

func writeFile(path string, data []byte) error {
	cleanPath := filepath.Clean(path)
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return fmt.Errorf("invalid output path: %s", path)
	}
	return os.WriteFile(cleanPath, data, 0644)
}

func formatHTML(userInput string) string {
	return "<div>" + html.EscapeString(userInput) + "</div>"
}
