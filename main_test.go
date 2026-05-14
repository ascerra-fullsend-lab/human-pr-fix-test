package main

import (
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
}

func TestAllowedCommands(t *testing.T) {
	t.Run("allowed commands are in allowlist", func(t *testing.T) {
		expected := []string{"ls", "cat", "echo", "date", "whoami"}
		for _, cmd := range expected {
			if !allowedCommands[cmd] {
				t.Errorf("expected %q to be allowed", cmd)
			}
		}
	})

	t.Run("arbitrary commands are not allowed", func(t *testing.T) {
		dangerous := []string{"rm", "sh", "bash", "curl", "wget", "dd"}
		for _, cmd := range dangerous {
			if allowedCommands[cmd] {
				t.Errorf("expected %q to NOT be allowed", cmd)
			}
		}
	})
}
