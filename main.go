package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// commandPolicy defines whether a command accepts arguments and how to validate them.
type commandPolicy struct {
	argsAllowed  bool
	validateArgs func(args []string) error
}

// allowedCommands defines the set of commands that are permitted to execute,
// along with their argument validation policies.
var allowedCommands = map[string]commandPolicy{
	"ls":     {argsAllowed: true, validateArgs: validatePathArgs},
	"cat":    {argsAllowed: true, validateArgs: validatePathArgs},
	"echo":   {argsAllowed: true, validateArgs: nil}, // echo allows any args
	"date":   {argsAllowed: false},
	"whoami": {argsAllowed: false},
}

// validatePathArgs checks that all arguments are safe relative paths (no absolute paths,
// no path traversal, no flag injection).
func validatePathArgs(args []string) error {
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("flags not allowed: %s", arg)
		}
		clean := filepath.Clean(arg)
		if filepath.IsAbs(clean) {
			return fmt.Errorf("absolute path not allowed: %s", arg)
		}
		if strings.HasPrefix(clean, "..") {
			return fmt.Errorf("path traversal not allowed: %s", arg)
		}
	}
	return nil
}

// maxResponseBytes caps the HTTP response body size to prevent memory exhaustion.
const maxResponseBytes = 10 * 1024 * 1024 // 10 MB

// httpTimeout sets the maximum duration for HTTP requests.
const httpTimeout = 30 * time.Second

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <command> <output-file> <data-url>\n", os.Args[0])
		os.Exit(1)
	}

	userCmd := os.Args[1]
	parts := strings.Fields(userCmd)
	if len(parts) == 0 {
		fmt.Fprintf(os.Stderr, "command must not be empty\n")
		os.Exit(1)
	}

	policy, ok := allowedCommands[parts[0]]
	if !ok {
		fmt.Fprintf(os.Stderr, "command not allowed: %s\n", parts[0])
		os.Exit(1)
	}

	args := parts[1:]
	if len(args) > 0 && !policy.argsAllowed {
		fmt.Fprintf(os.Stderr, "command %q does not accept arguments\n", parts[0])
		os.Exit(1)
	}
	if policy.validateArgs != nil && len(args) > 0 {
		if err := policy.validateArgs(args); err != nil {
			fmt.Fprintf(os.Stderr, "invalid arguments for %s: %v\n", parts[0], err)
			os.Exit(1)
		}
	}

	out, err := exec.Command(parts[0], args...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "command error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprint(os.Stderr, string(out))

	dataURL := os.Args[3]
	if !strings.HasPrefix(dataURL, "https://") {
		fmt.Fprintf(os.Stderr, "data URL must use HTTPS: %s\n", dataURL)
		os.Exit(1)
	}

	client := &http.Client{Timeout: httpTimeout}
	resp, err := client.Get(dataURL)
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
	fmt.Fprint(os.Stderr, string(body))

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

	// Resolve symlinks in the parent directory to prevent symlink-based escapes.
	dir := filepath.Dir(cleanPath)
	if dir != "." {
		resolvedDir, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return fmt.Errorf("cannot resolve output directory: %v", err)
		}
		// After resolving, ensure we haven't escaped via symlink.
		absResolved, err := filepath.Abs(resolvedDir)
		if err != nil {
			return fmt.Errorf("cannot resolve absolute path: %v", err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %v", err)
		}
		if !strings.HasPrefix(absResolved, cwd) {
			return fmt.Errorf("output path escapes working directory via symlink: %s", path)
		}
		cleanPath = filepath.Join(resolvedDir, filepath.Base(cleanPath))
	}

	return os.WriteFile(cleanPath, data, 0644)
}
