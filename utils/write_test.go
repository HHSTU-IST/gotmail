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

// TestWriteFilePrivateFollowsDanglingSymlinkChain covers the dangling end of the
// symlink contract.
//
// EvalSymlinks needs the far end to exist, so a chain ending in a file that is
// not there yet falls through to symlinkTarget's own resolution. Reading a
// single hop there replaces the next link in the chain with a real file — a
// dotfile manager's link quietly turned into a copy — where the caller asked to
// write through it and create the target.
func TestWriteFilePrivateFollowsDanglingSymlinkChain(t *testing.T) {
	dir := t.TempDir()

	final := filepath.Join(dir, "final.json")
	hop3 := filepath.Join(dir, "c.json")
	hop2 := filepath.Join(dir, "b.json")
	hop1 := filepath.Join(dir, "a.json")

	// Relative targets, so resolving against the link's own directory is
	// exercised as well.
	plantSymlink(t, "final.json", hop3)
	plantSymlink(t, "c.json", hop2)
	plantSymlink(t, "b.json", hop1)

	if err := writeFilePrivate(hop1, []byte(`{"chain":true}`), true); err != nil {
		t.Fatalf("writeFilePrivate: %v", err)
	}

	content, err := os.ReadFile(final)
	if err != nil {
		t.Fatalf("expected the write to create %q: %v", final, err)
	}
	if string(content) != `{"chain":true}` {
		t.Errorf("Expected the data at %q, got %q", final, content)
	}

	for _, link := range []string{hop1, hop2, hop3} {
		info, err := os.Lstat(link)
		if err != nil {
			t.Fatalf("lstat %q: %v", link, err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("Expected %q to survive as a symlink, it was replaced by a real file", link)
		}
	}
}

// TestWriteFilePrivateCleansStaleTempFiles covers the residue a killed process
// leaves behind. SIGKILL skips the deferred removal in replaceFilePrivate, and
// for the accounts file the directory in question is $HOME, so a later run has
// to collect it.
//
// The near-misses matter as much as the target: an entry young enough to belong
// to a write in flight must survive, and so must a directory or an unrelated
// file that merely shares the prefix.
func TestWriteFilePrivateCleansStaleTempFiles(t *testing.T) {
	dir := t.TempDir()

	stale := filepath.Join(dir, tempFilePrefix+"abandoned")
	if err := os.WriteFile(stale, []byte("half-written"), 0o600); err != nil {
		t.Fatalf("planting a stale scratch file: %v", err)
	}
	aged := time.Now().Add(-2 * tempFileMaxAge)
	if err := os.Chtimes(stale, aged, aged); err != nil {
		t.Fatalf("ageing the stale scratch file: %v", err)
	}

	fresh := filepath.Join(dir, tempFilePrefix+"inflight")
	if err := os.WriteFile(fresh, []byte("live"), 0o600); err != nil {
		t.Fatalf("planting a fresh scratch file: %v", err)
	}

	decoyDir := filepath.Join(dir, tempFilePrefix+"dir")
	if err := os.Mkdir(decoyDir, 0o700); err != nil {
		t.Fatalf("planting a directory under the prefix: %v", err)
	}

	unrelated := filepath.Join(dir, "keep.txt")
	if err := os.WriteFile(unrelated, []byte("keep"), 0o600); err != nil {
		t.Fatalf("planting an unrelated file: %v", err)
	}

	if err := writeFilePrivate(filepath.Join(dir, "target.json"), []byte(`{"ok":true}`), true); err != nil {
		t.Fatalf("writeFilePrivate: %v", err)
	}

	if _, err := os.Lstat(stale); !os.IsNotExist(err) {
		t.Errorf("Expected the stale scratch file to be collected, lstat returned %v", err)
	}
	for _, keep := range []string{fresh, decoyDir, unrelated} {
		if _, err := os.Lstat(keep); err != nil {
			t.Errorf("Expected %q to be left alone: %v", keep, err)
		}
	}
}

// TestWriteFilePrivateReportsReadOnlyDirectory pins the diagnostic half of the
// atomic-replace tradeoff: the directory, not the file, is what has to be
// writable, and the error says so rather than reporting a bare permission
// failure against a file the user can see is writable.
//
// Whether the platform enforces it is asked rather than assumed. Windows puts
// only a read-only attribute on a directory and a root-owned job bypasses the
// check, so the test attempts the write and reports when it cannot exercise
// the case here.
func TestWriteFilePrivateReportsReadOnlyDirectory(t *testing.T) {
	dir := t.TempDir()

	target := filepath.Join(dir, "account.json")
	if err := os.WriteFile(target, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatalf("planting the target file: %v", err)
	}

	if err := os.Chmod(dir, 0o500); err != nil {
		t.Skipf("cannot make the directory read-only here: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o700) })

	err := writeFilePrivate(target, []byte(`{"new":true}`), true)
	if err == nil {
		t.Skip("this platform still allows creating files in a read-only directory; the diagnostic cannot be exercised here")
	}

	if !strings.Contains(err.Error(), dir) {
		t.Errorf("Expected the error to name the directory %q, got %v", dir, err)
	}
	if !strings.Contains(err.Error(), "write permission on that directory") {
		t.Errorf("Expected the error to explain the directory requirement, got %v", err)
	}
}

// TestWriteFilePrivateHardLinkKeepsOldContents pins a documented tradeoff
// rather than a defect. The write replaces path as a directory entry, so a hard
// link to the same file under another name keeps the contents it had.
//
// Both halves cannot hold at once: writing through the link is an in-place
// write, which is the thing the atomic replace exists to avoid, and the link
// count needed to spot the case beforehand lives in a platform-specific stat
// structure. If this test fails because the write started following hard links,
// the comment on writeFilePrivate has to change with it.
func TestWriteFilePrivateHardLinkKeepsOldContents(t *testing.T) {
	dir := t.TempDir()

	original := filepath.Join(dir, "original.json")
	if err := os.WriteFile(original, []byte(`{"old":true}`), 0o600); err != nil {
		t.Fatalf("planting the original file: %v", err)
	}

	link := filepath.Join(dir, "linked.json")
	if err := os.Link(original, link); err != nil {
		t.Skipf("hard links unavailable here: %v", err)
	}

	if err := writeFilePrivate(link, []byte(`{"new":true}`), true); err != nil {
		t.Fatalf("writeFilePrivate: %v", err)
	}

	written, err := os.ReadFile(link)
	if err != nil {
		t.Fatalf("reading %q: %v", link, err)
	}
	if string(written) != `{"new":true}` {
		t.Errorf("Expected the write to land at %q, got %q", link, written)
	}

	sibling, err := os.ReadFile(original)
	if err != nil {
		t.Fatalf("reading %q: %v", original, err)
	}
	if string(sibling) != `{"old":true}` {
		t.Errorf("Expected the other name to keep %q, got %q", `{"old":true}`, sibling)
	}
}
