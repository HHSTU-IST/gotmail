package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// MailManager mail manager
type MailManager struct {
	db     *Database
	client *http.Client
	color  *Color
}

// NewMailManager creates new mail manager
func NewMailManager(dataPath string) *MailManager {
	return &MailManager{
		db: NewDatabase(dataPath),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		color: &Color{},
	}
}

// CreateAccount creates new account
func (m *MailManager) CreateAccount() error {
	spinner := NewSpinner("creating...")
	spinner.Start()
	defer spinner.Stop()

	// Read account data
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	// Check if maximum accounts limit reached (optional, can be adjusted)
	accounts := m.db.GetData()
	if len(accounts) >= 10 {
		fmt.Printf("%s\n", m.color.Red("Maximum 10 accounts allowed"))
		return nil
	}

	// Get available domain
	domain, err := m.getDomain()
	if err != nil {
		return fmt.Errorf("failed to get domain: %w", err)
	}

	// Generate random email and password
	emailLocalPart, err := GenerateRandomString(7)
	if err != nil {
		return fmt.Errorf("failed to generate email address: %w", err)
	}
	email := fmt.Sprintf("%s@%s", emailLocalPart, domain)

	password, err := GenerateRandomString(10)
	if err != nil {
		return fmt.Errorf("failed to generate password: %w", err)
	}

	// Create account
	accountID, err := m.createAccountAPI(email, password)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	// Get JWT token
	token, err := m.getToken(email, password)
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// Build account information
	account := &Account{
		ID:        accountID,
		Address:   email,
		Password:  password,
		Token:     TokenData{Token: token},
		CreatedAt: time.Now(),
	}

	// Copy email to clipboard
	if err := Copy(email); err != nil {
		fmt.Printf("Warning: failed to copy email to clipboard: %v\n", err)
	}

	// Save account data
	if err := m.db.AddAccount(account); err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}
	if err := m.db.Write(); err != nil {
		return fmt.Errorf("failed to save account: %w", err)
	}

	fmt.Printf("\r%s: %s (ID: %s)\n", m.color.Blue("Account created"), m.color.Underline(m.color.Green(email)), m.color.Green(account.ID))
	return nil
}

// ExportAccount exports account data to specified folder
func (m *MailManager) ExportAccount(exportFolder string) error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	// Ensure export directory exists. The export holds plaintext credentials,
	// so a directory this program creates defaults to owner-only too.
	if err := os.MkdirAll(exportFolder, 0700); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	exportPath := filepath.Join(exportFolder, "accounts.json")

	// Read the original account file
	originalData, err := os.ReadFile(m.db.dataPath)
	if err != nil {
		return fmt.Errorf("failed to read original account file: %w", err)
	}

	// Write to export path
	if err := writeFilePrivate(exportPath, originalData, false); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("Account data exported to: %s\n", exportPath)
	return nil
}

// ExportAccountByID exports specific account data to specified path
func (m *MailManager) ExportAccountByID(accountID string, exportFolder string) error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	account := m.db.GetAccount(accountID)
	if account == nil {
		return fmt.Errorf("account with ID %s not found", accountID)
	}

	// Ensure export directory exists. The export holds plaintext credentials,
	// so a directory this program creates defaults to owner-only too.
	if err := os.MkdirAll(exportFolder, 0700); err != nil {
		return fmt.Errorf("failed to create export directory: %w", err)
	}

	exportPath := filepath.Join(exportFolder, fmt.Sprintf("account_%s.json", accountID))

	// Create single account data for export
	singleAccountData := map[string]*Account{accountID: account}
	exportData, err := json.MarshalIndent(singleAccountData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal account data: %w", err)
	}

	// Write to export path
	if err := writeFilePrivate(exportPath, exportData, false); err != nil {
		return fmt.Errorf("failed to write export file: %w", err)
	}

	fmt.Printf("Account %s exported to: %s\n", accountID, exportPath)
	return nil
}

// GetAllAccountsJSON returns all accounts data as JSON string
func (m *MailManager) GetAllAccountsJSON() (string, error) {
	if err := m.db.Read(); err != nil {
		return "", fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		return "[]", nil
	}

	// Convert accounts to JSON
	jsonData, err := json.MarshalIndent(accounts, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal accounts data: %w", err)
	}

	return string(jsonData), nil
}

// ListAccounts lists all accounts
func (m *MailManager) ListAccounts() error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		fmt.Printf("%s\n", m.color.Red("No accounts found"))
		return nil
	}

	fmt.Println(FormatAccountList(accounts))
	return nil
}

// FetchMessagesByAccountID fetches messages for specific account
func (m *MailManager) FetchMessagesByAccountID(accountID string) ([]Message, error) {
	spinner := NewSpinner("fetching messages...")
	spinner.Start()
	defer spinner.Stop()

	if err := m.db.Read(); err != nil {
		return nil, fmt.Errorf("failed to read database: %w", err)
	}

	account := m.db.GetAccount(accountID)
	if account == nil {
		return nil, fmt.Errorf("account with ID %s not found", accountID)
	}

	messages, err := m.fetchMessagesAPI(account.Token.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Printf("\r\033[K%s\n", m.color.Red("No Emails"))
		return nil, nil
	}

	return messages, nil
}

// ShowAccountDetails shows details for specific account
func (m *MailManager) ShowAccountDetails(accountID string) error {
	spinner := NewSpinner("fetching details...")
	spinner.Start()
	defer spinner.Stop()

	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	account := m.db.GetAccount(accountID)
	if account == nil {
		return fmt.Errorf("account with ID %s not found", accountID)
	}

	// Use existing account data instead of API call for basic info
	fmt.Printf("\n    Account ID: %s\n    Email: %s\n    Created: %s\n",
		m.color.Green(account.ID),
		m.color.Underline(m.color.Green(account.Address)),
		m.color.Green(account.CreatedAt.Format("2006-01-02 15:04:05")))

	return nil
}

// DeleteAccountByID deletes specific account by ID
func (m *MailManager) DeleteAccountByID(accountID string) error {
	spinner := NewSpinner("deleting account...")
	spinner.Start()
	defer spinner.Stop()

	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	account := m.db.GetAccount(accountID)
	if account == nil {
		return fmt.Errorf("account with ID %s not found", accountID)
	}

	if err := m.deleteAccountAPI(account.ID, account.Token.Token); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	if err := m.db.DeleteAccount(accountID); err != nil {
		return fmt.Errorf("failed to delete account data: %w", err)
	}

	if err := m.db.Write(); err != nil {
		return fmt.Errorf("failed to save database: %w", err)
	}

	fmt.Printf("%s\n", m.color.Blue("Account deleted"))
	return nil
}

// FetchMessages fetches email messages for an interactively selected account
// when --id is not given.
func (m *MailManager) FetchMessages() ([]Message, error) {
	if err := m.db.Read(); err != nil {
		return nil, fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		fmt.Printf("%s\n", m.color.Red("No accounts found"))
		return nil, nil
	}

	// Ask which account to use. Taking the first value a map iteration yields
	// is taking one at random, which used to send the request to an arbitrary
	// account. Selection runs before the spinner starts so the prompt is not
	// overwritten by the animation.
	account, err := SelectAccount(accounts)
	if err != nil {
		return nil, err
	}

	spinner := NewSpinner("fetching...")
	spinner.Start()
	defer spinner.Stop()

	messages, err := m.fetchMessagesAPI(account.Token.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 {
		fmt.Printf("\r\033[K%s\n", m.color.Red("No Emails"))
		return nil, nil
	}

	return messages, nil
}

// DeleteAccount deletes the first account (backward compatibility)
func (m *MailManager) DeleteAccount() error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		fmt.Printf("%s\n", m.color.Red("No accounts found"))
		return nil
	}

	// Use interactive selection like SelectAccount
	account, err := SelectAccount(accounts)
	if err != nil {
		return err
	}

	spinner := NewSpinner("deleting...")
	spinner.Start()
	defer spinner.Stop()

	// Find the account ID from the map
	var accountID string
	for id, acc := range accounts {
		if acc == account {
			accountID = id
			break
		}
	}

	if err := m.deleteAccountAPI(account.ID, account.Token.Token); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	if err := m.db.DeleteAccount(accountID); err != nil {
		return fmt.Errorf("failed to delete account data: %w", err)
	}

	if err := m.db.Write(); err != nil {
		return fmt.Errorf("failed to save database: %w", err)
	}

	fmt.Printf("%s\n", m.color.Blue("Account deleted"))
	return nil
}

// ShowDetails shows details for an interactively selected account.
//
// NOTE: currently unreachable — `gotmail show` without --id prints every
// account through GetAllAccountsJSON instead. Kept in sync with FetchMessages
// so it cannot reintroduce random selection if it is ever wired up again.
func (m *MailManager) ShowDetails() error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		fmt.Printf("%s\n", m.color.Red("No accounts found"))
		return nil
	}

	account, err := SelectAccount(accounts)
	if err != nil {
		return err
	}

	spinner := NewSpinner("fetching details...")
	spinner.Start()
	defer spinner.Stop()

	// Use existing account data instead of API call for basic info
	fmt.Printf("\n    Account ID: %s\n    Email: %s\n    Created: %s\n",
		m.color.Green(account.ID),
		m.color.Underline(m.color.Green(account.Address)),
		m.color.Green(account.CreatedAt.Format("2006-01-02 15:04:05")))

	return nil
}

// OpenEmail opens a specified email for an interactively selected account when
// --id is not given.
func (m *MailManager) OpenEmail(emailIndex int) error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	accounts := m.db.GetData()
	if len(accounts) == 0 {
		return fmt.Errorf("no accounts found")
	}

	// Same reasoning as FetchMessages: without --id the account must be chosen
	// deliberately, not taken from a random map iteration.
	account, err := SelectAccount(accounts)
	if err != nil {
		return err
	}

	spinner := NewSpinner("opening...")
	spinner.Start()
	defer spinner.Stop()

	messages, err := m.fetchMessagesAPI(account.Token.Token)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 || emailIndex < 1 || emailIndex > len(messages) {
		return fmt.Errorf("invalid email index")
	}

	mailToOpen := messages[emailIndex-1]
	emailDetail, err := m.getEmailDetailAPI(mailToOpen.ID, account.Token.Token)
	if err != nil {
		return fmt.Errorf("failed to get email detail: %w", err)
	}

	if len(emailDetail.HTML) == 0 {
		fmt.Printf("%s\n", m.color.Red("No HTML content found"))
		return nil
	}

	// Write HTML file. Every render gets a fresh 0600 file in the user cache
	// directory; see writeEmailHTML for why it is not deleted afterwards.
	emailFilePath, err := writeEmailHTML(emailDetail.HTML[0])
	if err != nil {
		return fmt.Errorf("failed to write email file: %w", err)
	}

	// Open file in browser
	if err := openInBrowser(emailFilePath); err != nil {
		return fmt.Errorf("failed to open email in browser: %w", err)
	}

	return nil
}

// OpenEmailByAccountID opens specified email for specific account
func (m *MailManager) OpenEmailByAccountID(accountID string, emailIndex int) error {
	if err := m.db.Read(); err != nil {
		return fmt.Errorf("failed to read database: %w", err)
	}

	account := m.db.GetAccount(accountID)
	if account == nil {
		return fmt.Errorf("account with ID %s not found", accountID)
	}

	spinner := NewSpinner("opening...")
	spinner.Start()
	defer spinner.Stop()

	messages, err := m.fetchMessagesAPI(account.Token.Token)
	if err != nil {
		return fmt.Errorf("failed to fetch messages: %w", err)
	}

	if len(messages) == 0 || emailIndex < 1 || emailIndex > len(messages) {
		return fmt.Errorf("invalid email index")
	}

	mailToOpen := messages[emailIndex-1]
	emailDetail, err := m.getEmailDetailAPI(mailToOpen.ID, account.Token.Token)
	if err != nil {
		return fmt.Errorf("failed to get email detail: %w", err)
	}

	if len(emailDetail.HTML) == 0 {
		fmt.Printf("%s\n", m.color.Red("No HTML content found"))
		return nil
	}

	// Write HTML file. Every render gets a fresh 0600 file in the user cache
	// directory; see writeEmailHTML for why it is not deleted afterwards.
	emailFilePath, err := writeEmailHTML(emailDetail.HTML[0])
	if err != nil {
		return fmt.Errorf("failed to write email file: %w", err)
	}

	// Open file in browser
	if err := openInBrowser(emailFilePath); err != nil {
		return fmt.Errorf("failed to open email in browser: %w", err)
	}

	return nil
}

// getDomain gets available domain
func (m *MailManager) getDomain() (string, error) {
	resp, err := m.client.Get("https://api.mail.tm/domains?page=1")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", buildAPIStatusError("get domain", resp)
	}

	var domainResp DomainResponse
	if err := json.NewDecoder(resp.Body).Decode(&domainResp); err != nil {
		return "", err
	}

	if len(domainResp.HydraMember) == 0 {
		return "", fmt.Errorf("no domains available")
	}

	return domainResp.HydraMember[0].Domain, nil
}

// createAccountAPI calls API to create account
func (m *MailManager) createAccountAPI(email, password string) (string, error) {
	payload := map[string]string{
		"address":  email,
		"password": password,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := m.client.Post("https://api.mail.tm/accounts", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", buildAPIStatusError("create account", resp)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.ID == "" {
		return "", fmt.Errorf("create account response missing id")
	}

	return result.ID, nil
}

// getToken gets JWT token
func (m *MailManager) getToken(email, password string) (string, error) {
	payload := map[string]string{
		"address":  email,
		"password": password,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := m.client.Post("https://api.mail.tm/token", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", buildAPIStatusError("get token", resp)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Token == "" {
		return "", fmt.Errorf("token response missing token")
	}

	return result.Token, nil
}

// fetchMessagesAPI fetches email messages API
func (m *MailManager) fetchMessagesAPI(token string) ([]Message, error) {
	req, err := http.NewRequest("GET", "https://api.mail.tm/messages", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, buildAPIStatusError("fetch messages", resp)
	}

	var msgResp MessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&msgResp); err != nil {
		return nil, err
	}

	return msgResp.HydraMember, nil
}

// deleteAccountAPI deletes account API
func (m *MailManager) deleteAccountAPI(accountID, token string) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("https://api.mail.tm/accounts/%s", accountID), nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return buildAPIStatusError("delete account", resp)
	}

	return nil
}

// getAccountDetailsAPI gets account details API
func (m *MailManager) getAccountDetailsAPI(accountID, token string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.mail.tm/accounts/%s", accountID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, buildAPIStatusError("get account details", resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

// getEmailDetailAPI gets email detail API
func (m *MailManager) getEmailDetailAPI(messageID, token string) (*EmailDetail, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.mail.tm/messages/%s", messageID), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := m.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, buildAPIStatusError("get email detail", resp)
	}

	var result EmailDetail
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// emailHTMLMaxAge is how long a rendered email is kept before the next render
// prunes it. It only needs to outlast the browser handing the file to a viewer.
const emailHTMLMaxAge = time.Hour

// emailCacheDir returns the per-user directory that holds rendered emails.
//
// This used to be <executable dir>/../data/email.html, which fails when the
// install prefix is read-only (for example a Homebrew binary in
// /opt/homebrew/bin) and leaves mail content inside the installation
// directory. The user cache directory is always writable and is the
// conventional home for data the program can regenerate.
func emailCacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		// Deliberately no fallback to os.TempDir(). That directory is
		// world-writable, so on a shared machine an attacker could pre-create
		// the path component used below; because they would own the parent
		// directory they could then swap the rendered file before the browser
		// reads it, turning a mail preview into script execution under
		// file://. A missing cache directory is an environment problem worth
		// reporting rather than papering over.
		return "", fmt.Errorf("cannot determine a user cache directory: %w", err)
	}

	dir := filepath.Join(base, "gotmail", "email")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	return dir, nil
}

// writeEmailHTML renders html into a fresh file in the cache directory and
// returns its path.
//
// Every render gets its own file: os.CreateTemp opens with O_CREATE|O_EXCL, so
// a symbolic link planted at a predictable name cannot redirect the write, and
// two concurrent renders cannot clobber each other. The file is created 0600
// because email HTML is attacker-influenced content.
//
// The file is deliberately not deleted afterwards. openInBrowser launches the
// browser with exec.Start and does not wait for it, so removing the file
// straight away would race the viewer. Instead stale renders are pruned once
// they are older than emailHTMLMaxAge, which bounds how long message content
// stays on disk; the rest is documented in `gotmail help open`.
func writeEmailHTML(html string) (string, error) {
	dir, err := emailCacheDir()
	if err != nil {
		return "", err
	}

	// Best effort: a stale file that cannot be removed must not fail this render.
	cleanStaleEmailHTML(dir)

	f, err := os.CreateTemp(dir, "email-*.html")
	if err != nil {
		return "", fmt.Errorf("failed to create email file: %w", err)
	}
	path := f.Name()

	if _, err := f.WriteString(html); err != nil {
		f.Close()
		os.Remove(path)
		return "", fmt.Errorf("failed to write email file: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("failed to write email file: %w", err)
	}

	return path, nil
}

// cleanStaleEmailHTML removes rendered emails older than emailHTMLMaxAge.
//
// Errors are deliberately ignored. The cache is best-effort, and on Windows a
// file the browser still holds open cannot be removed at all; leaving it until
// the next render is the correct outcome there.
func cleanStaleEmailHTML(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	cutoff := time.Now().Add(-emailHTMLMaxAge)
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, "email-") || !strings.HasSuffix(name, ".html") {
			continue
		}

		info, err := entry.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}

		os.Remove(filepath.Join(dir, name))
	}
}

// openInBrowser opens file in browser
func openInBrowser(filePath string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", filePath}
	case "darwin":
		cmd = "open"
		args = []string{filePath}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{filePath}
	}

	return exec.Command(cmd, args...).Start()
}

func buildAPIStatusError(action string, resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	msg := extractAPIErrorMessage(body)
	if msg == "" {
		return fmt.Errorf("%s failed with status %d", action, resp.StatusCode)
	}
	return fmt.Errorf("%s failed with status %d: %s", action, resp.StatusCode, msg)
}

func extractAPIErrorMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err == nil {
		for _, key := range []string{"message", "detail", "hydra:description", "title"} {
			if raw, ok := payload[key]; ok {
				if msg, ok := raw.(string); ok {
					msg = strings.TrimSpace(msg)
					if msg != "" {
						return msg
					}
				}
			}
		}
	}

	return trimmed
}
