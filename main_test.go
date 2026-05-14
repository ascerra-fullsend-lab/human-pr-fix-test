package main

import (
	"net"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFile(t *testing.T) {
	t.Run("valid relative path", func(t *testing.T) {
		dir := t.TempDir()
		// Change to temp dir so relative paths resolve there
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		err := writeFile("output.txt", []byte("hello"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "output.txt"))
		if err != nil {
			t.Fatalf("failed to read written file: %v", err)
		}
		if string(data) != "hello" {
			t.Errorf("got %q, want %q", string(data), "hello")
		}
	})

	t.Run("rejects absolute path", func(t *testing.T) {
		err := writeFile("/etc/passwd", []byte("bad"))
		if err == nil {
			t.Fatal("expected error for absolute path, got nil")
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		err := writeFile("../../etc/passwd", []byte("bad"))
		if err == nil {
			t.Fatal("expected error for path traversal, got nil")
		}
	})

	t.Run("rejects dotdot prefix", func(t *testing.T) {
		err := writeFile("../secret", []byte("bad"))
		if err == nil {
			t.Fatal("expected error for .. prefix, got nil")
		}
	})

	t.Run("rejects symlink escape", func(t *testing.T) {
		dir := t.TempDir()
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)
		os.Chdir(dir)

		// Create a symlink pointing outside the working directory
		os.Symlink("/tmp", filepath.Join(dir, "escape"))

		err := writeFile("escape/pwned.txt", []byte("bad"))
		if err == nil {
			t.Fatal("expected error for symlink escape, got nil")
			// Clean up in case it was written
			os.Remove("/tmp/pwned.txt")
		}
	})
}

func TestAllowedCommands(t *testing.T) {
	t.Run("allowed commands are in allowlist", func(t *testing.T) {
		expected := []string{"ls", "cat", "echo", "date", "whoami"}
		for _, cmd := range expected {
			if _, ok := allowedCommands[cmd]; !ok {
				t.Errorf("expected %q to be allowed", cmd)
			}
		}
	})

	t.Run("arbitrary commands are not allowed", func(t *testing.T) {
		dangerous := []string{"rm", "sh", "bash", "curl", "wget", "dd"}
		for _, cmd := range dangerous {
			if _, ok := allowedCommands[cmd]; ok {
				t.Errorf("expected %q to NOT be allowed", cmd)
			}
		}
	})
}

func TestValidatePathArgs(t *testing.T) {
	t.Run("rejects absolute paths", func(t *testing.T) {
		err := validatePathArgs([]string{"/etc/shadow"})
		if err == nil {
			t.Fatal("expected error for absolute path arg")
		}
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		err := validatePathArgs([]string{"../../etc/passwd"})
		if err == nil {
			t.Fatal("expected error for path traversal arg")
		}
	})

	t.Run("rejects flags", func(t *testing.T) {
		err := validatePathArgs([]string{"-la"})
		if err == nil {
			t.Fatal("expected error for flag arg")
		}
	})

	t.Run("allows safe relative paths", func(t *testing.T) {
		err := validatePathArgs([]string{"file.txt", "subdir/file.txt"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestCommandPolicies(t *testing.T) {
	t.Run("date rejects arguments", func(t *testing.T) {
		policy := allowedCommands["date"]
		if policy.argsAllowed {
			t.Fatal("date should not allow arguments")
		}
	})

	t.Run("whoami rejects arguments", func(t *testing.T) {
		policy := allowedCommands["whoami"]
		if policy.argsAllowed {
			t.Fatal("whoami should not allow arguments")
		}
	})

	t.Run("cat validates path args", func(t *testing.T) {
		policy := allowedCommands["cat"]
		if !policy.argsAllowed {
			t.Fatal("cat should allow arguments")
		}
		if policy.validateArgs == nil {
			t.Fatal("cat should have argument validation")
		}
		// cat should reject absolute paths
		err := policy.validateArgs([]string{"/etc/shadow"})
		if err == nil {
			t.Fatal("cat should reject absolute path arguments")
		}
	})

	t.Run("echo allows any arguments", func(t *testing.T) {
		policy := allowedCommands["echo"]
		if !policy.argsAllowed {
			t.Fatal("echo should allow arguments")
		}
		// echo has no validateArgs, so any args are fine
		if policy.validateArgs != nil {
			t.Fatal("echo should not restrict arguments")
		}
	})
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		private bool
	}{
		{"loopback", "127.0.0.1", true},
		{"private 10.x", "10.0.0.1", true},
		{"private 172.16.x", "172.16.0.1", true},
		{"private 192.168.x", "192.168.1.1", true},
		{"link-local", "169.254.1.1", true},
		{"public", "8.8.8.8", false},
		{"public 2", "93.184.216.34", false},
		{"ipv6 loopback", "::1", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %s", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.private)
			}
		})
	}
}

func TestWriteFileSymlinkBoundary(t *testing.T) {
	// Regression test: ensure prefix matching doesn't allow "/application"
	// when cwd is "/app" — uses filepath.Rel instead of HasPrefix.
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)
	os.Chdir(dir)

	// Create a symlink pointing outside the working directory
	os.Symlink("/tmp", filepath.Join(dir, "escape"))

	err := writeFile("escape/pwned.txt", []byte("bad"))
	if err == nil {
		t.Fatal("expected error for symlink escape, got nil")
		os.Remove("/tmp/pwned.txt")
	}
}
