package utils_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/ivaquero/gotmail/utils"
)

// captureOutput captures stdout output for testing
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestShowHelp(t *testing.T) {
	fmt.Println("=== Test ShowHelp function ===")

	output := captureOutput(func() {
		utils.ShowHelp()
	})

	// Test that output contains expected content
	expectedContents := []string{
		"Your Temporary Email Accounts Manager",
		"Usage:",
		"new",
		"ls",
		"msg",
		"del",
		"show",
		"open",
		"export",
		"help",
		"--id",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("ShowHelp output missing expected content: %s", expected)
		}
	}

	// Test that output doesn't contain unexpected content
	if strings.Contains(output, "Unknown command") {
		t.Error("ShowHelp output should not contain 'Unknown command'")
	}

	fmt.Println("ShowHelp test passed!")
}

func TestShowCommandHelp(t *testing.T) {
	fmt.Println("=== Test ShowCommandHelp function ===")

	testCases := []struct {
		command            string
		expectedContents   []string
		unexpectedContents []string
	}{
		{
			command: "new",
			expectedContents: []string{
				"Create a new account",
				"gotmail new",
				"Creates a new temporary email account",
			},
			unexpectedContents: []string{"Unknown command"},
		},
		{
			command: "ls",
			expectedContents: []string{
				"List all accounts",
				"gotmail ls",
				"Displays all stored email accounts",
			},
			unexpectedContents: []string{"Unknown command"},
		},
		{
			command: "msg",
			expectedContents: []string{
				"Fetch and list messages",
				"gotmail msg",
				"--id <account_id>",
				"Pick an account interactively",
			},
			unexpectedContents: []string{"Unknown command", "default account"},
		},
		{
			command: "del",
			expectedContents: []string{
				"Delete account",
				"gotmail del",
				"--id <account_id>",
				"Pick an account interactively",
			},
			unexpectedContents: []string{"Unknown command", "default account"},
		},
		{
			command: "show",
			expectedContents: []string{
				"Show account details or all accounts",
				"gotmail show",
				"--id <account_id>",
				"JSON format",
			},
			unexpectedContents: []string{"Unknown command"},
		},
		{
			command: "open",
			expectedContents: []string{
				"Open specific email in browser",
				"gotmail open <number>",
				"--id <account_id>",
				"Pick an account, open its first message",
			},
			unexpectedContents: []string{"Unknown command", "default account"},
		},
		{
			command: "export",
			expectedContents: []string{
				"Export account data to specified folder",
				"gotmail export <folder>",
				"--id <account_id>",
				"/tmp/backup",
			},
			unexpectedContents: []string{"Unknown command"},
		},
		{
			command: "help",
			expectedContents: []string{
				"Show help information",
				"gotmail help [command]",
				"Show general help",
			},
			unexpectedContents: []string{"Unknown command"},
		},
		{
			command: "invalid",
			expectedContents: []string{
				"Unknown command: invalid",
				"Available commands:",
				"new, ls, msg, del, show, open, export, help",
			},
			unexpectedContents: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.command, func(t *testing.T) {
			output := captureOutput(func() {
				utils.ShowCommandHelp(tc.command)
			})

			// Check expected contents
			for _, expected := range tc.expectedContents {
				if !strings.Contains(output, expected) {
					t.Errorf("ShowCommandHelp(%s) output missing expected content: %s", tc.command, expected)
				}
			}

			// Check unexpected contents
			for _, unexpected := range tc.unexpectedContents {
				if unexpected != "" && strings.Contains(output, unexpected) {
					t.Errorf("ShowCommandHelp(%s) output contains unexpected content: %s", tc.command, unexpected)
				}
			}
		})
	}

	fmt.Println("ShowCommandHelp test passed!")
}

func TestHelpFunctionsOutput(t *testing.T) {
	fmt.Println("=== Test help functions output format ===")

	// Test ShowHelp output format
	showHelpOutput := captureOutput(func() {
		utils.ShowHelp()
	})

	// Verify output has proper structure
	lines := strings.Split(strings.TrimSpace(showHelpOutput), "\n")
	if len(lines) < 10 {
		t.Errorf("ShowHelp output too short, expected at least 10 lines, got %d", len(lines))
	}

	// Test ShowCommandHelp output format for a specific command
	showCommandHelpOutput := captureOutput(func() {
		utils.ShowCommandHelp("new")
	})

	commandLines := strings.Split(strings.TrimSpace(showCommandHelpOutput), "\n")
	if len(commandLines) < 5 {
		t.Errorf("ShowCommandHelp output too short, expected at least 5 lines, got %d", len(commandLines))
	}

	// Verify that command help includes usage, description, etc.
	if !strings.Contains(showCommandHelpOutput, "Usage:") {
		t.Error("ShowCommandHelp output should include 'Usage:' section")
	}

	if !strings.Contains(showCommandHelpOutput, "Description:") {
		t.Error("ShowCommandHelp output should include 'Description:' section")
	}

	fmt.Println("Help functions output format test passed!")
}

// TestHelpExamplesUseUsableAccountIDs guards a defect found in review: every
// example ID in the help text was "abc123", which the CLI itself rejects.
// validateAccountID (main.go:14) requires 10-50 alphanumeric characters, so an
// example that fails it is worse than no example at all.
func TestHelpExamplesUseUsableAccountIDs(t *testing.T) {
	fmt.Println("=== Test help examples use usable account IDs ===")

	commands := []string{"new", "ls", "msg", "del", "show", "open", "export", "help"}
	for _, command := range commands {
		output := captureOutput(func() {
			utils.ShowCommandHelp(command)
		})

		for _, line := range strings.Split(output, "\n") {
			// Only invocation lines are examples. Prose such as "When --id is
			// omitted" also mentions --id but carries no example value.
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "gotmail ") {
				continue
			}

			idx := strings.Index(line, "--id ")
			if idx < 0 {
				continue
			}

			// The value runs to end of line or to the first explanatory comment.
			value := strings.TrimSpace(line[idx+len("--id "):])
			if cut := strings.Index(value, " "); cut >= 0 {
				value = value[:cut]
			}

			// Usage lines document the placeholder rather than a concrete value.
			if strings.HasPrefix(value, "<") {
				continue
			}

			if len(value) < 10 || len(value) > 50 {
				t.Errorf("gotmail help %s: example ID %q is %d characters, the CLI requires 10-50",
					command, value, len(value))
			}
			for _, r := range value {
				if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
					t.Errorf("gotmail help %s: example ID %q contains %q, the CLI allows letters and digits only",
						command, value, r)
					break
				}
			}
		}
	}

	fmt.Println("Help example account ID test passed!")
}
