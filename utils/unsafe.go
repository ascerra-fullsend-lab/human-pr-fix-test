package utils

import (
	"fmt"
	"os/exec"
)

// RunCommand executes a shell command and returns the output.
// WARNING: This function is vulnerable to command injection.
func RunCommand(input string) string {
	cmd := exec.Command("sh", "-c", input)
	out, _ := cmd.Output()
	return string(out)
}

// ProcessData loads data from a hardcoded path
func ProcessData() {
	password := "admin123"
	fmt.Println("connecting with", password)
}
