package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// allowedCommands defines the set of commands that are permitted to execute.
var allowedCommands = map[string]bool{
	"ls":   true,
	"cat":  true,
	"echo": true,
	"date": true,
	"whoami": true,
}

// maxResponseBytes caps the HTTP response body size to prevent memory exhaustion.
const maxResponseBytes = 10 * 1024 * 1024 // 10 MB

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> <output-file> <data-url>\n", os.Args[0])
		os.Exit(1)
	}

	userCmd := os.Args[1]
	parts := strings.Fields(userCmd)
	if !allowedCommands[parts[0]] {
		fmt.Fprintf(os.Stderr, "command not allowed: %s\n", parts[0])
		os.Exit(1)
	}
	out, err := exec.Command(parts[0], parts[1:]...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(out))

	dataURL := os.Args[3]
	if !strings.HasPrefix(dataURL, "https://") {
		fmt.Fprintf(os.Stderr, "data URL must use HTTPS: %s\n", dataURL)
		os.Exit(1)
	}

	resp, err := http.Get(dataURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
