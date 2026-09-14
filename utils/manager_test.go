package utils

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestManager(rt roundTripFunc) *MailManager {
	return &MailManager{
		client: &http.Client{
			Timeout:   time.Second,
			Transport: rt,
		},
		color: &Color{},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestCreateAccountAPIStatusError(t *testing.T) {
	manager := newTestManager(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/accounts" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(http.StatusUnprocessableEntity, `{"message":"address is invalid"}`), nil
	})

	id, err := manager.createAccountAPI("bad", "password")
	if err == nil {
		t.Fatal("expected error for non-201 response")
	}
	if id != "" {
		t.Fatalf("expected empty id, got %q", id)
	}
	if !strings.Contains(err.Error(), "status 422") {
		t.Fatalf("expected status in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "address is invalid") {
		t.Fatalf("expected API message in error, got: %v", err)
	}
}

func TestCreateAccountAPIMissingID(t *testing.T) {
	manager := newTestManager(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusCreated, `{"address":"x@example.com"}`), nil
	})

	_, err := manager.createAccountAPI("x@example.com", "password")
	if err == nil {
		t.Fatal("expected error when id is missing")
	}
	if !strings.Contains(err.Error(), "missing id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetTokenMissingToken(t *testing.T) {
	manager := newTestManager(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/token" {
			t.Fatalf("unexpected path: %s", req.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"foo":"bar"}`), nil
	})

	_, err := manager.getToken("x@example.com", "password")
	if err == nil {
		t.Fatal("expected error when token is missing")
	}
	if !strings.Contains(err.Error(), "missing token") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchMessagesAPIStatusError(t *testing.T) {
	manager := newTestManager(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusUnauthorized, `{"hydra:description":"JWT Token not found"}`), nil
	})

	_, err := manager.fetchMessagesAPI("bad-token")
	if err == nil {
		t.Fatal("expected error for unauthorized response")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected status code in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "JWT Token not found") {
		t.Fatalf("expected hydra message in error, got: %v", err)
	}
}

// redirectUserCacheDir points os.UserCacheDir at a directory inside the test's
// temp dir, so these tests never create or delete files in the real user cache.
func redirectUserCacheDir(t *testing.T) string {
	t.Helper()

	base := t.TempDir()
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", base)
	case "darwin":
		t.Setenv("HOME", base)
	default:
		t.Setenv("XDG_CACHE_HOME", base)
	}

	dir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("os.UserCacheDir failed after redirection: %v", err)
	}
	return dir
}

// TestEmailCacheDirFailsClosedWithoutUserCacheDir covers the fallback that was
// removed here: rendering used to fall back to os.TempDir(), which is
// world-writable, so on a shared machine an attacker could pre-create the
// directory that receives the rendered mail and swap the file before the
// browser read it.
func TestEmailCacheDirFailsClosedWithoutUserCacheDir(t *testing.T) {
	switch runtime.GOOS {
	case "windows":
		t.Setenv("LocalAppData", "")
	case "darwin":
		t.Setenv("HOME", "")
	default:
		t.Setenv("XDG_CACHE_HOME", "")
		t.Setenv("HOME", "")
	}

	if _, err := os.UserCacheDir(); err == nil {
		t.Skip("os.UserCacheDir still resolves here, so the failure path is unreachable")
	}

	if _, err := emailCacheDir(); err == nil {
		t.Error("emailCacheDir must report an error rather than fall back to a world-writable directory")
	}
}

// TestWriteEmailHTMLLivesInUserCacheDir covers P1-4: the rendered email used to
// be written to <executable dir>/../data/email.html, which fails outright when
// the install prefix is read-only (for example a Homebrew binary in
// /opt/homebrew/bin).
func TestWriteEmailHTMLLivesInUserCacheDir(t *testing.T) {
	cacheDir := redirectUserCacheDir(t)

	path, err := writeEmailHTML("<p>hello</p>")
	if err != nil {
		t.Fatalf("writeEmailHTML returned an error: %v", err)
	}

	wantDir := filepath.Join(cacheDir, "gotmail", "email")
	if got := filepath.Dir(path); got != wantDir {
		t.Errorf("Expected the email file in %q, got %q", wantDir, got)
	}
	if name := filepath.Base(path); !strings.HasPrefix(name, "email-") || !strings.HasSuffix(name, ".html") {
		t.Errorf("Expected a name like email-*.html, got %q", name)
	}

	// Guard against a regression to the old executable-relative location.
	if exe, exeErr := os.Executable(); exeErr == nil {
		if exeDir := filepath.Dir(exe); strings.HasPrefix(path, exeDir) {
			t.Errorf("Email file must not be written next to the executable (%q), got %q", exeDir, path)
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Expected the rendered email to be readable: %v", err)
	}
	if string(content) != "<p>hello</p>" {
		t.Errorf("Expected the rendered HTML back, got %q", content)
	}

	// Email HTML is attacker-influenced content, so it must not be world-readable.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %q: %v", path, err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("Expected the rendered email to be 0600, got %#o", perm)
		}
	}
}

// TestWriteEmailHTMLKeepsRendersApart covers the two cache properties the
// security review asked for: each render gets its own file, so a second open
// cannot clobber the first while the browser still displays it, and renders
// older than emailHTMLMaxAge are pruned so message content does not accumulate
// forever in the cache.
func TestWriteEmailHTMLKeepsRendersApart(t *testing.T) {
	cacheDir := redirectUserCacheDir(t)

	first, err := writeEmailHTML("<p>one</p>")
	if err != nil {
		t.Fatalf("first writeEmailHTML: %v", err)
	}
	second, err := writeEmailHTML("<p>two</p>")
	if err != nil {
		t.Fatalf("second writeEmailHTML: %v", err)
	}
	if first == second {
		t.Fatalf("Expected a distinct file per render, both were %q", first)
	}
	if _, err := os.Stat(first); err != nil {
		t.Errorf("A render fresh enough to still be on screen must survive the next one: %v", err)
	}

	// Plant a render old enough to be stale, so the test does not wait an hour.
	dir := filepath.Join(cacheDir, "gotmail", "email")
	stale := filepath.Join(dir, "email-stale.html")
	if err := os.WriteFile(stale, []byte("<p>old</p>"), 0o600); err != nil {
		t.Fatalf("planting stale render: %v", err)
	}
	old := time.Now().Add(-2 * emailHTMLMaxAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatalf("ageing stale render: %v", err)
	}

	if _, err := writeEmailHTML("<p>three</p>"); err != nil {
		t.Fatalf("third writeEmailHTML: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("Expected the stale render %q to be pruned, stat err = %v", stale, err)
	}
	if _, err := os.Stat(first); err != nil {
		t.Errorf("A render younger than emailHTMLMaxAge must be kept: %v", err)
	}
}

// TestWriteEmailHTMLDoesNotFollowPlantedSymlink is a regression guard for the
// fixed name this used to write (<cache>/gotmail/email.html): a symlink planted
// there must not be handed the rendered message.
func TestWriteEmailHTMLDoesNotFollowPlantedSymlink(t *testing.T) {
	cacheDir := redirectUserCacheDir(t)
	dir := filepath.Join(cacheDir, "gotmail", "email")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("creating cache directory: %v", err)
	}

	victim := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(victim, []byte("untouched"), 0o600); err != nil {
		t.Fatalf("planting victim file: %v", err)
	}
	if err := os.Symlink(victim, filepath.Join(dir, "email.html")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	path, err := writeEmailHTML("<p>secret</p>")
	if err != nil {
		t.Fatalf("writeEmailHTML: %v", err)
	}

	content, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("reading victim file: %v", err)
	}
	if string(content) != "untouched" {
		t.Errorf("A planted symlink must not receive the rendered email, victim is now %q", content)
	}
	if name := filepath.Base(path); name == "email.html" {
		t.Errorf("Expected a fresh unpredictable name, got %q", name)
	}
}
