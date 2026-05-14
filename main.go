package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
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
// Note: argument parsing uses strings.Fields which splits on whitespace — quoted
// arguments (e.g., echo "hello world") are not supported.
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

	dataURL := os.Args[3]
	if !strings.HasPrefix(dataURL, "https://") {
		fmt.Fprintf(os.Stderr, "data URL must use HTTPS: %s\n", dataURL)
		os.Exit(1)
	}

	if err := validateURL(dataURL); err != nil {
		fmt.Fprintf(os.Stderr, "URL validation error: %v\n", err)
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

	// Combine command output and fetched data into the output file.
	combined := append(out, body...)
	if err := writeFile(os.Args[2], combined); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
}

// isPrivateIP returns true if the IP is in a private, loopback, or link-local range.
func isPrivateIP(ip net.IP) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range privateRanges {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// validateURL resolves the hostname and rejects URLs pointing to private/link-local IPs
// to prevent SSRF attacks against internal services.
func validateURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}
	host := parsed.Hostname()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return fmt.Errorf("DNS resolution failed for %s: %v", host, err)
	}
	for _, ipAddr := range ips {
		if isPrivateIP(ipAddr.IP) {
			return fmt.Errorf("URL resolves to private/link-local IP %s — rejected to prevent SSRF", ipAddr.IP)
		}
	}
	return nil
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
		// Use filepath.Rel to verify the resolved path stays within cwd.
		// This avoids the prefix-matching pitfall where "/app" matches "/application".
		absResolved, err := filepath.Abs(resolvedDir)
		if err != nil {
			return fmt.Errorf("cannot resolve absolute path: %v", err)
		}
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %v", err)
		}
		rel, err := filepath.Rel(cwd, absResolved)
		if err != nil {
			return fmt.Errorf("cannot compute relative path: %v", err)
		}
		if strings.HasPrefix(rel, "..") {
			return fmt.Errorf("output path escapes working directory via symlink: %s", path)
		}
		cleanPath = filepath.Join(resolvedDir, filepath.Base(cleanPath))
	}

	return os.WriteFile(cleanPath, data, 0644)
}
