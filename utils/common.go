package utils

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Account account data structure
type Account struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Password  string    `json:"password"`
	Token     TokenData `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}

// TokenData JWT token data structure
type TokenData struct {
	Token string `json:"token"`
}

// Domain API returned domain data structure
type Domain struct {
	Domain string `json:"domain"`
}

// DomainResponse API domain response
type DomainResponse struct {
	HydraMember []Domain `json:"hydra:member"`
}

// Message email message data structure
type Message struct {
	ID   string `json:"id"`
	From struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"from"`
	To []struct {
		Address string `json:"address"`
		Name    string `json:"name"`
	} `json:"to"`
}

// MessageResponse API message response
type MessageResponse struct {
	HydraMember []Message `json:"hydra:member"`
}

// EmailDetail email detail data structure
type EmailDetail struct {
	ID      string   `json:"id"`
	HTML    []string `json:"html"`
	Subject string   `json:"subject"`
}

// Database database operation structure for multiple accounts
type Database struct {
	dataPath string
	accounts map[string]*Account // Key: account ID, Value: account data
}

// NewDatabase creates new database instance
func NewDatabase(dataPath string) *Database {
	return &Database{
		dataPath: dataPath,
		accounts: make(map[string]*Account),
	}
}

// Read reads accounts data
func (db *Database) Read() error {
	data, err := os.ReadFile(db.dataPath)
	if err != nil {
		if os.IsNotExist(err) {
			db.accounts = make(map[string]*Account)
			return nil
		}
		return fmt.Errorf("failed to read accounts file: %w", err)
	}

	var accounts map[string]*Account
	if err := json.Unmarshal(data, &accounts); err != nil {
		// Try to read as single account (backward compatibility)
		var singleAccount Account
		if err := json.Unmarshal(data, &singleAccount); err != nil {
			return fmt.Errorf("failed to unmarshal accounts data: %w", err)
		}
		// Convert single account to map format
		db.accounts = map[string]*Account{singleAccount.ID: &singleAccount}
		return nil
	}

	db.accounts = accounts
	return nil
}

// Write writes accounts data
func (db *Database) Write() error {
	if db.accounts == nil {
		return nil
	}

	data, err := json.MarshalIndent(db.accounts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal accounts data: %w", err)
	}

	// Ensure directory exists. It holds credentials, so a directory this
	// program creates is owner-only. MkdirAll leaves an existing directory's
	// mode alone, so this only applies when it actually creates one.
	dir := filepath.Dir(db.dataPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := writeFilePrivate(db.dataPath, data, true); err != nil {
		return fmt.Errorf("failed to write accounts file: %w", err)
	}

	return nil
}

// writeFilePrivate writes data to path with owner-only permissions (0600).
//
// Both modes go through replaceFilePrivate, so the bytes land in a fresh file
// that is created 0600 and then renamed into place. That buys two properties at
// once: os.WriteFile applies perm only when it creates a file, so writing in
// place would leave a pre-existing world-readable file readable, and a rename
// is atomic, so a crash part-way through cannot leave a truncated accounts file
// behind. The accounts file holds plaintext passwords and JWTs.
//
// allowSymlink decides what happens when path is a symbolic link:
//
//   - true  — write to the link's target and leave the link alone. Used for
//     ~/.gotmail.json, because dotfile managers (stow, chezmoi) legitimately
//     symlink a dotfile into place and replacing the link with a real file
//     would quietly break that setup.
//   - false — replace whatever sits at path. Used for export files, whose final
//     path component the program picks itself, so a link planted in the
//     destination directory must not be able to redirect the write onto some
//     other file the user can write to.
//
// Two consequences of writing through a temporary file plus a rename are worth
// recording, because neither can be removed without giving up the atomicity
// above, and a reader who hits either one should not read it as a defect:
//
//   - The directory has to be writable, not just the file being replaced. A
//     read-only directory therefore fails here even when path itself would
//     accept a write. replaceFilePrivate says as much in its error, since a
//     bare "permission denied" against a file the user can see is writable
//     looks like a bug in this program.
//   - path is replaced as a directory entry, so a hard link to the same file
//     under another name keeps the old contents. Hard links and atomic
//     replacement are mutually exclusive: writing through the link is exactly
//     the in-place write that the properties above rule out. The link count
//     also lives in a platform-specific stat structure, so the case cannot even
//     be detected portably before writing. It is stated here rather than worked
//     around.
func writeFilePrivate(path string, data []byte, allowSymlink bool) error {
	target := path
	if allowSymlink {
		target = symlinkTarget(path)
	}
	return replaceFilePrivate(target, data)
}

// symlinkTarget returns the file a write to path should land on, leaving any
// symlink in place.
//
// EvalSymlinks resolves a chain of links only when the far end exists, so a
// dangling link is followed one hop at a time instead. Looping rather than
// reading a single hop matters for a chain whose last link points at nothing:
// resolving just one hop would leave the second link in the chain to be
// replaced by a real file, quietly dropping it, where the caller meant to write
// through it. The hop limit matches the kernel's ELOOP threshold, so a genuine
// loop stops instead of spinning.
func symlinkTarget(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}

	const maxHops = 40
	for hop := 0; hop < maxHops; hop++ {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return path
		}

		target, err := os.Readlink(path)
		if err != nil {
			return path
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
	return path
}

// tempFilePrefix names the scratch file replaceFilePrivate renames into place.
// A dot prefix keeps it out of a plain directory listing, where it would
// otherwise appear next to the user's files even when nothing has gone wrong.
const tempFilePrefix = ".gotmail-tmp-"

// tempFileMaxAge is how old a scratch file must be before a later run treats it
// as abandoned. The window in which a live scratch file exists is the gap
// between CreateTemp and Rename — microseconds — so an hour cannot collide with
// a write in flight while still collecting files left behind by a process that
// was killed. SIGKILL skips the deferred cleanup below, and for the accounts
// file the directory in question is $HOME.
const tempFileMaxAge = time.Hour

// replaceFilePrivate writes data to a fresh file next to path and renames it
// into place.
//
// os.CreateTemp opens with O_CREATE|O_EXCL, so it can neither follow nor reuse
// a symbolic link planted at path, and rename replaces the destination entry
// itself rather than writing through it. Together they close the window
// between a check and the write that a plain lstat-then-write would leave open.
func replaceFilePrivate(path string, data []byte) error {
	dir := filepath.Dir(path)

	// Collect scratch files an earlier run could not clean up. Doing this before
	// the write rather than after means the sweep also runs on a process that
	// dies during this very write, which is the case that produces the residue.
	sweepTempFiles(dir)

	tmp, err := os.CreateTemp(dir, tempFilePrefix+"*")
	if err != nil {
		// Name the directory and the reason. "permission denied" pointing at a
		// file the user can write is the confusing half of the tradeoff
		// documented on writeFilePrivate above.
		return fmt.Errorf("cannot create a temporary file in %s (replacing %s atomically needs write permission on that directory, not only on the file): %w",
			dir, path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

// sweepTempFiles removes scratch files that an earlier run left in dir after
// being killed before it could rename or delete them.
//
// Best effort on purpose: failing to tidy up must not fail the write that
// follows, because performing that write is what the caller asked for. An entry
// qualifies only if its name carries the scratch prefix and it is a regular
// file of at least tempFileMaxAge, so a directory or a symbolic link planted
// under that name is skipped — remove would follow neither, but a link that
// disappears because it happened to match a prefix is still an unwanted
// surprise, and a directory is not something this program creates.
func sweepTempFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-tempFileMaxAge)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), tempFilePrefix) {
			continue
		}

		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.ModTime().After(cutoff) {
			continue
		}

		os.Remove(filepath.Join(dir, entry.Name()))
	}
}

// GetData gets all accounts data
func (db *Database) GetData() map[string]*Account {
	return db.accounts
}

// GetAccount gets specific account by ID
func (db *Database) GetAccount(accountID string) *Account {
	if db.accounts == nil {
		return nil
	}
	return db.accounts[accountID]
}

// SetAccount sets specific account data
func (db *Database) SetAccount(accountID string, data *Account) {
	if db.accounts == nil {
		db.accounts = make(map[string]*Account)
	}
	db.accounts[accountID] = data
}

// AddAccount adds new account
func (db *Database) AddAccount(data *Account) error {
	if db.accounts == nil {
		db.accounts = make(map[string]*Account)
	}
	if _, exists := db.accounts[data.ID]; exists {
		return fmt.Errorf("account with ID %s already exists", data.ID)
	}
	db.accounts[data.ID] = data
	return nil
}

// DeleteAccount deletes specific account
func (db *Database) DeleteAccount(accountID string) error {
	if db.accounts == nil {
		return fmt.Errorf("no accounts found")
	}
	if _, exists := db.accounts[accountID]; !exists {
		return fmt.Errorf("account with ID %s not found", accountID)
	}
	delete(db.accounts, accountID)
	return nil
}

// GetAllAccountIDs gets all account IDs
func (db *Database) GetAllAccountIDs() []string {
	if db.accounts == nil {
		return []string{}
	}
	ids := make([]string, 0, len(db.accounts))
	for id := range db.accounts {
		ids = append(ids, id)
	}
	return ids
}

// DeleteData deletes accounts data file
func (db *Database) DeleteData() error {
	if err := os.Remove(db.dataPath); err != nil {
		return fmt.Errorf("failed to delete accounts file: %w", err)
	}
	db.accounts = make(map[string]*Account)
	return nil
}

// GenerateRandomString generates a random string drawn from [a-z0-9].
//
// It returns an error rather than falling back to a predictable sequence. The
// old fallback built each character from charset[i%len(charset)], so a
// crypto/rand failure silently produced "abcdefg..." — a guessable password
// that was reported as a success. This function mints account credentials, so
// failing loudly is the only safe behaviour.
func GenerateRandomString(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("random string length must be positive, got %d", length)
	}

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	maxIndex := big.NewInt(int64(len(charset)))
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, maxIndex)
		if err != nil {
			return "", fmt.Errorf("failed to generate random string: %w", err)
		}
		b[i] = charset[num.Int64()]
	}
	return string(b), nil
}

// Spinner simple loading animation
type Spinner struct {
	message string
	done    chan bool
}

// NewSpinner creates new loading animation
func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan bool),
	}
}

// Start starts loading animation
func (s *Spinner) Start() {
	go func() {
		chars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Printf("\r%s\r", strings.Repeat(" ", len(s.message)+10))
				return
			default:
				fmt.Printf("\r%s %s", chars[i%len(chars)], s.message)
				time.Sleep(100 * time.Millisecond)
				i++
			}
		}
	}()
}

// Stop stops loading animation
func (s *Spinner) Stop() {
	close(s.done)
}

// Color simple color output functions
type Color struct{}

// colorEnabled reports whether colour should be emitted.
//
// It follows the NO_COLOR convention: colour is off when the variable is
// present and non-empty. LookupEnv rather than Getenv, because Getenv cannot
// distinguish "set to nothing" — which by that convention means "leave colour
// alone" — from "not set at all", and the two call for opposite behaviour.
//
// The variable is read per call rather than cached, so a program that sets it
// after start-up is still obeyed and tests can turn colour off around a single
// call instead of having to re-exec.
//
// Only colour is gated. The cursor and erase sequences the spinner and the
// "No Emails" line use (\r, \033[K) are terminal control rather than colour, so
// they are left alone; deciding whether to suppress those is the job of a
// non-interactive output mode, which NO_COLOR does not describe.
func colorEnabled() bool {
	value, ok := os.LookupEnv("NO_COLOR")
	return !ok || value == ""
}

// wrap applies an SGR code around text, or returns text untouched when colour
// is disabled.
func (c Color) wrap(code string, text string) string {
	if !colorEnabled() {
		return text
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", code, text)
}

// Red red output
func (c Color) Red(text string) string {
	return c.wrap("31", text)
}

// Green green output
func (c Color) Green(text string) string {
	return c.wrap("32", text)
}

// Blue blue output
func (c Color) Blue(text string) string {
	return c.wrap("34", text)
}

// Underline underline output
func (c Color) Underline(text string) string {
	return c.wrap("4", text)
}

// Helper functions for multi-account management

// ParseAccountID parses account ID from command arguments
func ParseAccountID(args []string) (string, bool) {
	for i, arg := range args {
		if arg == "--id" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// ParseEmailID parses email ID from command arguments
func ParseEmailID(args []string) (string, bool) {
	for i, arg := range args {
		if arg == "--email" && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// FormatAccountList formats account list for display
func FormatAccountList(accounts map[string]*Account) string {
	if len(accounts) == 0 {
		return "No accounts found"
	}

	var result strings.Builder
	color := &Color{}
	result.WriteString("Available accounts:\n")

	for id, account := range accounts {
		result.WriteString(fmt.Sprintf("  %s %s (created: %s)\n",
			color.Green(id),
			color.Underline(account.Address),
			account.CreatedAt.Format("2006-01-02 15:04:05")))
	}

	return result.String()
}

// SelectAccount prompts user to select an account if no ID provided
func SelectAccount(accounts map[string]*Account) (*Account, error) {
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no accounts found")
	}

	if len(accounts) == 1 {
		// Auto-select if only one account
		for _, account := range accounts {
			return account, nil
		}
	}

	// Multiple accounts - show list and ask for selection
	fmt.Println(FormatAccountList(accounts))
	fmt.Print("Enter account ID: ")

	var selectedID string
	fmt.Scanln(&selectedID)
	selectedID = strings.TrimSpace(selectedID)

	account, exists := accounts[selectedID]
	if !exists {
		return nil, fmt.Errorf("account with ID %s not found", selectedID)
	}

	return account, nil
}
