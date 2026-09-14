package utils_test

import (
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/ivaquero/gotmail/utils"
)

// clipboardCommand reports the utility utils.Copy shells out to on this
// platform, if any.
//
// It restates the dispatch in utils.Copy rather than asking it, on purpose.
// A probe routed through the code under test could not distinguish "this host
// has no working clipboard" from "utils.Copy is broken", and those two call for
// opposite answers: the first is a reason to skip, the second is the failure
// this test exists to catch.
func clipboardCommand() (string, bool) {
	switch runtime.GOOS {
	case "windows":
		return "clip", true
	case "darwin":
		return "pbcopy", true
	case "linux":
		for _, candidate := range []string{"xclip", "xsel"} {
			if _, err := exec.LookPath(candidate); err == nil {
				return candidate, true
			}
		}
	}
	return "", false
}

// clipboardUsable reports whether command can actually take a copy here.
//
// Knowing the platform is not enough. `clip` exists on Windows but fails
// without a usable console, and xclip needs a display, so the utility has to be
// run once to find out. The probe feeds a short string through a pipe the same
// way Copy does; that fits inside the pipe buffer, so a child that exits without
// draining stdin cannot block the copy goroutine.
func clipboardUsable(command string) error {
	probe := exec.Command(command)
	probe.Stdin = strings.NewReader("gotmail clipboard probe")
	return probe.Run()
}

// TestCopy checks whichever clipboard contract applies to this host, instead of
// assuming the platform decides it.
//
// Where the clipboard works, utils.Copy must succeed for every payload. Where
// it does not, utils.Copy must report the failure: a copy that did not happen
// while the program says it did is worse than an error, because the user finds
// out by pasting the wrong thing.
func TestCopy(t *testing.T) {
	command, found := clipboardCommand()
	if !found {
		t.Skipf("no clipboard utility for %s; there is nothing to copy with", runtime.GOOS)
	}

	probeErr := clipboardUsable(command)
	if probeErr != nil {
		t.Logf("%s is installed but unusable on this host: %v", command, probeErr)
	}

	testCases := []struct {
		name string
		data string
	}{
		{"English text", "Hello, World!"},
		{"Chinese text", "这是测试数据。"},
		{"Mixed text", "Hello, World! 这是测试数据。"},
		{"Special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"Multi-line text", "第一行\n第二行\n第三行"},
	}

	failures := 0

	for _, tc := range testCases {
		err := utils.Copy(tc.data)

		switch {
		case probeErr != nil && err == nil:
			t.Errorf("%s cannot take a copy here (%v), but Copy(%s) reported success", command, probeErr, tc.name)
			failures++
		case probeErr == nil && err != nil:
			t.Errorf("Copy(%s) failed although the clipboard works: %v", tc.name, err)
			failures++
		}
	}

	if probeErr != nil && failures == 0 {
		t.Skipf("%s is installed but unusable here (%v); Copy correctly reported the failure", command, probeErr)
	}
}
