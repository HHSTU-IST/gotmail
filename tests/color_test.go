package utils_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ivaquero/gotmail/utils"
)

func TestColor(t *testing.T) {
	fmt.Println("=== Test Color functions ===")

	// Pin NO_COLOR to empty rather than leaving it to the environment: an empty
	// value means "colour stays on" under the convention, so this states the
	// assumption the assertions below rest on instead of inheriting it.
	t.Setenv("NO_COLOR", "")

	color := &utils.Color{}

	// Test Red function
	redText := color.Red("test text")
	if !strings.Contains(redText, "\033[31m") {
		t.Error("Red function should contain ANSI red color code")
	}
	if !strings.Contains(redText, "\033[0m") {
		t.Error("Red function should contain ANSI reset code")
	}
	if !strings.Contains(redText, "test text") {
		t.Error("Red function should contain the original text")
	}

	// Test Green function
	greenText := color.Green("test text")
	if !strings.Contains(greenText, "\033[32m") {
		t.Error("Green function should contain ANSI green color code")
	}
	if !strings.Contains(greenText, "\033[0m") {
		t.Error("Green function should contain ANSI reset code")
	}
	if !strings.Contains(greenText, "test text") {
		t.Error("Green function should contain the original text")
	}

	// Test Blue function
	blueText := color.Blue("test text")
	if !strings.Contains(blueText, "\033[34m") {
		t.Error("Blue function should contain ANSI blue color code")
	}
	if !strings.Contains(blueText, "\033[0m") {
		t.Error("Blue function should contain ANSI reset code")
	}
	if !strings.Contains(blueText, "test text") {
		t.Error("Blue function should contain the original text")
	}

	// Test Underline function
	underlineText := color.Underline("test text")
	if !strings.Contains(underlineText, "\033[4m") {
		t.Error("Underline function should contain ANSI underline code")
	}
	if !strings.Contains(underlineText, "\033[0m") {
		t.Error("Underline function should contain ANSI reset code")
	}
	if !strings.Contains(underlineText, "test text") {
		t.Error("Underline function should contain the original text")
	}

	// Test empty string handling
	emptyRed := color.Red("")
	if emptyRed != "\033[31m\033[0m" {
		t.Error("Color functions should handle empty strings properly")
	}

	// Test special characters
	specialText := "!@#$%^&*()"
	specialRed := color.Red(specialText)
	if !strings.Contains(specialRed, specialText) {
		t.Error("Color functions should handle special characters properly")
	}

	fmt.Println("Color functions test passed!")
}

func TestColorReset(t *testing.T) {
	fmt.Println("=== Test color reset functionality ===")

	// See TestColor: the reset suffix only exists while colour is on.
	t.Setenv("NO_COLOR", "")

	color := &utils.Color{}

	// Test that all functions end with reset code
	testCases := []struct {
		name     string
		function func(string) string
	}{
		{"Red", color.Red},
		{"Green", color.Green},
		{"Blue", color.Blue},
		{"Underline", color.Underline},
	}

	for _, tc := range testCases {
		result := tc.function("test")
		if !strings.HasSuffix(result, "\033[0m") {
			t.Errorf("%s function should end with reset code", tc.name)
		}
	}

	fmt.Println("Color reset functionality test passed!")
}

// unsetEnv removes name for the duration of the test and restores whatever was
// there before.
//
// testing.T has Setenv but no Unsetenv, and the two are not interchangeable
// here: colourEnabled has a separate branch for "unset" and for "set to empty",
// so the test has to be able to reach both.
func unsetEnv(t *testing.T, name string) {
	t.Helper()

	previous, present := os.LookupEnv(name)
	if err := os.Unsetenv(name); err != nil {
		t.Fatalf("cannot clear %s: %v", name, err)
	}
	t.Cleanup(func() {
		if present {
			os.Setenv(name, previous)
			return
		}
		os.Unsetenv(name)
	})
}

// TestColorHonoursNoColor checks the three states of NO_COLOR separately.
//
// The convention disables colour when the variable is present and non-empty, so
// "set to empty" has to keep colour on. Reading it with Getenv would collapse
// that case into "unset" and silently ignore a caller who set it to nothing.
func TestColorHonoursNoColor(t *testing.T) {
	color := &utils.Color{}

	palette := []struct {
		name   string
		method func(string) string
	}{
		{"Red", color.Red},
		{"Green", color.Green},
		{"Blue", color.Blue},
		{"Underline", color.Underline},
	}

	t.Setenv("NO_COLOR", "1")
	for _, tc := range palette {
		if got := tc.method("test"); got != "test" {
			t.Errorf("with NO_COLOR=1, %s returned %q, want the text unchanged", tc.name, got)
		}
	}

	t.Setenv("NO_COLOR", "")
	for _, tc := range palette {
		if got := tc.method("test"); !strings.Contains(got, "\033[") {
			t.Errorf("with NO_COLOR empty, %s returned %q, want an ANSI sequence", tc.name, got)
		}
	}

	unsetEnv(t, "NO_COLOR")
	for _, tc := range palette {
		if got := tc.method("test"); !strings.Contains(got, "\033[") {
			t.Errorf("with NO_COLOR unset, %s returned %q, want an ANSI sequence", tc.name, got)
		}
	}
}
