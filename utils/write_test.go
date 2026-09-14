package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// plantSymlink creates a symlink at link pointing to target, skipping the test
// when the platform refuses to create one (unprivileged Windows without
// Developer Mode).
func plantSymlink(t *testing.T, target, link string) {
	t.Helper()

	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable on this platform: %v", err)
	}
}

// TestWriteFilePrivateReplacesPlantedSymlink covers the export path: its final
// component is a name the program picks itself, so a link planted there must
// not be able to redirect the write onto some other file the user can write to.
func TestWriteFilePrivateReplacesPlantedSymlink(t *testing.T) {
	dir := t.TempDir()

	victim := filepath.Join(dir, "victim.txt")
	if err := os.WriteFile(victim, []byte("untouched"), 0o600); err != nil {
		t.Fatalf("planting victim file: %v", err)
	}

	target := filepath.Join(dir, "export.json")
	plantSymlink(t, victim, target)

	if err := writeFilePrivate(target, []byte(`{"secret":true}`), false); err != nil {
		t.Fatalf("writeFilePrivate: %v", err)
	}

	content, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("reading victim file: %v", err)
	}
	if string(content) != "untouched" {
		t.Errorf("A planted symlink must not receive the write, victim is now %q", content)
	}

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("lstat %q: %v", target, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Errorf("Expected %q to be replaced by a regular file, it is still a symlink", target)
	}

	written, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading %q: %v", target, err)
	}
	if string(written) != `{"secret":true}` {
		t.Errorf("Expected the data to land at %q, got %q", target, written)
	}

	// An export holds plaintext credentials, so the replacement file must be
	// owner-only as well.
	if runtime.GOOS != "windows" {
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("Expected mode 0600 on %q, got %#o", target, perm)
		}
	}
}

// TestWriteFilePrivateFollowsSymlinkWhenAllowed locks in the other half of the
// contract: ~/.gotmail.json may legitimately be a symlink placed by a dotfile
// manager (stow, chezmoi), so that path must keep following the link instead of
// replacing it with a real file.
func TestWriteFilePrivateFollowsSymlinkWhenAllowed(t *testing.T) {
	dir := t.TempDir()

	real := filepath.Join(dir, "real.json")
	if err := os.WriteFile(real, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatalf("planting real file: %v", err)
	}

	link := filepath.Join(dir, "linked.json")
	plantSymlink(t, real, link)

	if err := writeFilePrivate(link, []byte(`{"new":true}`), true); err != nil {
		t.Fatalf("writeFilePrivate: %v", err)
	}

	content, err := os.ReadFile(real)
	if err != nil {
		t.Fatalf("reading real file: %v", err)
	}
	if string(content) != `{"new":true}` {
		t.Errorf("Expected the write to follow the link into %q, got %q", real, content)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat %q: %v", link, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("Expected %q to remain a symlink for dotfile managers", link)
	}

	// The target is replaced rather than written through, so a loose mode on
	// the previous target is dropped instead of inherited — the property the
	// old truncate-then-write could not offer.
	if runtime.GOOS != "windows" {
		if err := os.Chmod(real, 0o644); err != nil {
			t.Fatalf("relaxing the link target: %v", err)
		}
		if err := writeFilePrivate(link, []byte(`{"third":true}`), true); err != nil {
			t.Fatalf("writeFilePrivate after relaxing: %v", err)
		}

		after, err := os.Stat(real)
		if err != nil {
			t.Fatalf("stat %q: %v", real, err)
		}
		if perm := after.Mode().Perm(); perm != 0o600 {
			t.Errorf("Expected the link target to come back 0600, got %#o", perm)
		}

		content, err := os.ReadFile(real)
		if err != nil {
			t.Fatalf("reading %q: %v", real, err)
		}
		if string(content) != `{"third":true}` {
			t.Errorf("Expected the third write to land in %q, got %q", real, content)
		}
	}
}

// TestExportAccountDoesNotFollowPlantedSymlink is the end-to-end version of the
// first test: it goes through the public command path so the assertion covers
// the wiring, not just the helper.
func TestExportAccountDoesNotFollowPlantedSymlink(t *testing.T) {
	home := t.TempDir()
	manager := NewMailManager(filepath.Join(home, ".gotmail.json"))

	if err := manager.db.AddAccount(&Account{
		ID:        "export12345",
		Address:   "export@example.com",
		Password:  "plaintext-secret",
		Token:     TokenData{Token: "jwt"},
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("adding account: %v", err)
	}
	if err := manager.db.Write(); err != nil {
		t.Fatalf("writing database: %v", err)
	}

	exportDir := t.TempDir()
	victim := filepath.Join(t.TempDir(), "victim.env")
	if err := os.WriteFile(victim, []byte("SECRET=untouched"), 0o600); err != nil {
		t.Fatalf("planting victim file: %v", err)
	}
	plantSymlink(t, victim, filepath.Join(exportDir, "accounts.json"))

	if err := manager.ExportAccount(exportDir); err != nil {
		t.Fatalf("ExportAccount: %v", err)
	}

	content, err := os.ReadFile(victim)
	if err != nil {
		t.Fatalf("reading victim file: %v", err)
	}
	if string(content) != "SECRET=untouched" {
		t.Errorf("Export must not write through a planted symlink, victim is now %q", content)
	}

	exported, err := os.ReadFile(filepath.Join(exportDir, "accounts.json"))
	if err != nil {
		t.Fatalf("reading exported file: %v", err)
	}
	if !strings.Contains(string(exported), "export12345") {
		t.Errorf("Expected the exported account in %q, got %q", exportDir, exported)
	}
}
