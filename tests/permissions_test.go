package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ivaquero/gotmail/utils"
)

// TestDatabaseWriteRestrictsFilePermissions covers B-1: the accounts file
// holds plaintext passwords and JWTs, so it must not be group- or
// world-readable.
//
// Skipped on Windows, where Go's os.Chmod only toggles the read-only attribute
// and POSIX permission bits are not enforced; the CI runner is Linux, so this
// assertion is exercised there.
func TestDatabaseWriteRestrictsFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission bits are not enforced on Windows")
	}

	fmt.Println("=== Test accounts file permissions ===")

	tempDir, err := os.MkdirTemp("", "gotmail_test_perm")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testDataPath := filepath.Join(tempDir, "test_perm.json")
	db := utils.NewDatabase(testDataPath)

	if err := db.AddAccount(&utils.Account{
		ID:        "perm1234567",
		Address:   "perm@example.com",
		Password:  "plaintext-secret",
		Token:     utils.TokenData{Token: "jwt"},
		CreatedAt: time.Now(),
	}); err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	if err := db.Write(); err != nil {
		t.Fatalf("Failed to write database: %v", err)
	}

	info, err := os.Stat(testDataPath)
	if err != nil {
		t.Fatalf("Failed to stat database file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("Expected accounts file mode 0600, got %#o", got)
	}

	// A file left world-readable by an older version must be tightened on the
	// next write rather than kept as it is: os.WriteFile only applies perm when
	// it creates the file, so without the explicit chmod the old mode survives.
	if err := os.Chmod(testDataPath, 0o644); err != nil {
		t.Fatalf("Failed to relax permissions for the rewrite check: %v", err)
	}
	if err := db.Write(); err != nil {
		t.Fatalf("Failed to rewrite database: %v", err)
	}

	info, err = os.Stat(testDataPath)
	if err != nil {
		t.Fatalf("Failed to stat database file after rewrite: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("Expected accounts file mode 0600 after rewrite, got %#o", got)
	}

	fmt.Println("Accounts file permission test passed!")
}
