package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureOutput captures both stdout and stderr for testing.
//
// Error messages such as "Unknown command" are written to os.Stderr, so
// redirecting only os.Stdout (as this helper used to do) meant those
// assertions could never see the text they looked for.
func captureOutput(f func()) string {
	oldStdout, oldStderr := os.Stdout, os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w

	f()

	w.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestRunWithoutArguments(t *testing.T) {
	output := captureOutput(func() {
		run(nil)
	})

	if !strings.Contains(output, "Your Temporary Email Accounts Manager") {
		t.Error("Expected help message when no arguments provided")
	}
}

func TestRunWithHelpCommand(t *testing.T) {
	output := captureOutput(func() {
		run([]string{"help"})
	})

	if !strings.Contains(output, "Your Temporary Email Accounts Manager") {
		t.Error("Expected help message when 'help' command provided")
	}
}

func TestRunWithInvalidCommand(t *testing.T) {
	var code int
	output := captureOutput(func() {
		code = run([]string{"invalidcommand"})
	})

	if !strings.Contains(output, "Unknown command") {
		t.Error("Expected 'Unknown command' message when invalid command provided")
	}

	if !strings.Contains(output, "Your Temporary Email Accounts Manager") {
		t.Error("Expected help message when invalid command provided")
	}

	if code != 2 {
		t.Errorf("Expected exit code 2 for an invalid command, got %d", code)
	}
}

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		positional []string
		accountID  string
		wantErr    bool
	}{
		{
			name:      "no arguments",
			args:      nil,
			accountID: "",
		},
		{
			name:       "positional only",
			args:       []string{"3"},
			positional: []string{"3"},
		},
		{
			name:       "flag before positional",
			args:       []string{"--id", "abcdefghij", "3"},
			positional: []string{"3"},
			accountID:  "abcdefghij",
		},
		{
			name:       "flag after positional",
			args:       []string{"3", "--id", "abcdefghij"},
			positional: []string{"3"},
			accountID:  "abcdefghij",
		},
		{
			name:       "equals form after positional",
			args:       []string{"./out", "--id=abcdefghij"},
			positional: []string{"./out"},
			accountID:  "abcdefghij",
		},
		{
			name:       "short flag after positional",
			args:       []string{"./out", "-id", "abcdefghij"},
			positional: []string{"./out"},
			accountID:  "abcdefghij",
		},
		{
			name:       "several positionals keep their order",
			args:       []string{"3", "extra", "--id", "abcdefghij"},
			positional: []string{"3", "extra"},
			accountID:  "abcdefghij",
		},
		{
			name:    "missing flag value",
			args:    []string{"3", "--id"},
			wantErr: true,
		},
		{
			name:    "unknown flag",
			args:    []string{"3", "--bogus"},
			wantErr: true,
		},
		{
			name:    "empty equals value",
			args:    []string{"--id="},
			wantErr: true,
		},
		{
			name:    "empty separate value",
			args:    []string{"--id", ""},
			wantErr: true,
		},
		{
			name:    "empty short equals value",
			args:    []string{"-id="},
			wantErr: true,
		},
		{
			name:       "double dash stops flag parsing",
			args:       []string{"--", "3"},
			positional: []string{"3"},
		},
		{
			name:       "double dash keeps later flags positional",
			args:       []string{"--", "--id"},
			positional: []string{"--id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			positional, accountID, err := splitArgs(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for args %v, got none", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if accountID != tt.accountID {
				t.Errorf("accountID = %q, want %q", accountID, tt.accountID)
			}
			if len(positional) != len(tt.positional) {
				t.Fatalf("positional = %v, want %v", positional, tt.positional)
			}
			for i := range positional {
				if positional[i] != tt.positional[i] {
					t.Errorf("positional[%d] = %q, want %q", i, positional[i], tt.positional[i])
				}
			}
		})
	}
}

func TestRunWithMalformedAccountID(t *testing.T) {
	var code int
	output := captureOutput(func() {
		// "zz" is rejected by validateAccountID before any file access, so this
		// stays a pure in-process check.
		code = run([]string{"del", "--id", "zz"})
	})

	if !strings.Contains(output, "Invalid account ID") {
		t.Errorf("Expected 'Invalid account ID' message, got %q", output)
	}

	if code != 2 {
		t.Errorf("Expected exit code 2 for a malformed account ID, got %d", code)
	}
}
