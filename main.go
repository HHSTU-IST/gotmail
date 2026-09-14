package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ivaquero/gotmail/utils"
)

// validateAccountID validates account ID format
func validateAccountID(accountID string) error {
	if accountID == "" {
		return fmt.Errorf("account ID cannot be empty")
	}
	// Account ID should be alphanumeric with length between 10-50 characters
	if len(accountID) < 10 || len(accountID) > 50 {
		return fmt.Errorf("account ID length must be between 10 and 50 characters")
	}
	// Simple alphanumeric validation
	matched, err := regexp.MatchString("^[a-zA-Z0-9]+$", accountID)
	if err != nil {
		return fmt.Errorf("failed to validate account ID format: %w", err)
	}
	if !matched {
		return fmt.Errorf("account ID can only contain letters and numbers")
	}
	return nil
}

// validateExportPath validates export folder path
func validateExportPath(path string) error {
	if path == "" {
		return fmt.Errorf("export path cannot be empty")
	}
	// Check if path is absolute or relative
	if filepath.IsAbs(path) {
		// For absolute paths, check if parent directory exists
		parent := filepath.Dir(path)
		if _, err := os.Stat(parent); err != nil {
			return fmt.Errorf("parent directory does not exist: %s", parent)
		}
	}
	return nil
}

// splitArgs separates the --id/-id flag from positional arguments.
//
// The flag package stops parsing as soon as it meets the first non-flag
// argument, so `gotmail open 3 --id abc123` silently dropped --id and acted on
// the wrong account. Pulling the flag out up front makes its position
// irrelevant. Unknown flags are still rejected.
func splitArgs(args []string) (positional []string, accountID string, err error) {
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--":
			// "--" ends flag parsing: everything after it is positional, which
			// is how the flag package this replaced behaved.
			positional = append(positional, args[i+1:]...)
			return positional, accountID, nil
		case arg == "--id" || arg == "-id":
			if i+1 >= len(args) {
				return nil, "", fmt.Errorf("flag needs an argument: %s", arg)
			}
			if args[i+1] == "" {
				return nil, "", fmt.Errorf("flag needs a non-empty argument: %s", arg)
			}
			accountID = args[i+1]
			i++
		case strings.HasPrefix(arg, "--id="):
			accountID = strings.TrimPrefix(arg, "--id=")
			if accountID == "" {
				return nil, "", fmt.Errorf("flag needs a non-empty argument: --id")
			}
		case strings.HasPrefix(arg, "-id="):
			accountID = strings.TrimPrefix(arg, "-id=")
			if accountID == "" {
				return nil, "", fmt.Errorf("flag needs a non-empty argument: -id")
			}
		case len(arg) > 1 && arg[0] == '-':
			return nil, "", fmt.Errorf("flag provided but not defined: %s", arg)
		default:
			positional = append(positional, arg)
		}
	}
	return positional, accountID, nil
}

func main() {
	os.Exit(run(os.Args[1:]))
}

// run executes a single gotmail invocation and returns the process exit code:
// 0 on success, 1 on runtime failure, 2 on usage error.
func run(args []string) int {
	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to get home directory: %v\n", err)
		return 1
	}

	// Set data file path to ~/.gotmail.json
	dataPath := filepath.Join(homeDir, ".gotmail.json")

	// Create mail manager
	mailManager := utils.NewMailManager(dataPath)

	// If no arguments, show help
	if len(args) == 0 {
		utils.ShowHelp()
		fmt.Println("\nTip: Use 'gotmail help <command>' for detailed command information")
		return 0
	}

	command := args[0]

	// Extract --id before dispatching, so it is honoured no matter where it
	// sits relative to positional arguments.
	positional, accountID, err := splitArgs(args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		utils.ShowHelp()
		return 2
	}
	hasAccountID := accountID != ""

	switch command {
	case "new":
		if err := mailManager.CreateAccount(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating account: %v\n", err)
			return 1
		}
		fmt.Println("Account created successfully!")
		return 0

	case "ls":
		if err := mailManager.ListAccounts(); err != nil {
			fmt.Fprintf(os.Stderr, "Error listing accounts: %v\n", err)
			return 1
		}
		return 0

	case "msg":
		var messages []utils.Message
		var err error

		if hasAccountID {
			if err := validateAccountID(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid account ID: %v\n", err)
				return 2
			}
			messages, err = mailManager.FetchMessagesByAccountID(accountID)
		} else {
			messages, err = mailManager.FetchMessages()
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching messages: %v\n", err)
			return 1
		}
		if messages != nil {
			if len(messages) == 0 {
				fmt.Println("No messages found")
			} else {
				fmt.Printf("Found %d message(s):\n", len(messages))
				for i, msg := range messages {
					fmt.Printf("  %d. ID: %s, From: %s\n", i+1, msg.ID, msg.From)
				}
			}
		}
		return 0

	case "del":
		if hasAccountID {
			// Validate account ID format
			if err := validateAccountID(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid account ID: %v\n", err)
				return 2
			}
			if err := mailManager.DeleteAccountByID(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting account: %v\n", err)
				return 1
			}
		} else {
			if err := mailManager.DeleteAccount(); err != nil {
				fmt.Fprintf(os.Stderr, "Error deleting account: %v\n", err)
				return 1
			}
		}
		return 0

	case "show":
		if hasAccountID {
			// 指定 id 时，显示 accounts.json 中的 id 对应信息
			if err := validateAccountID(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid account ID: %v\n", err)
				return 2
			}
			if err := mailManager.ShowAccountDetails(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Error showing account details: %v\n", err)
				return 1
			}
		} else {
			// 未指定 id 时，打印整个 accounts.json
			jsonData, err := mailManager.GetAllAccountsJSON()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error getting accounts JSON: %v\n", err)
				return 1
			}

			fmt.Println(jsonData)
			fmt.Println("Account data displayed successfully!")
		}
		return 0

	case "open":
		if len(positional) < 1 {
			fmt.Fprintf(os.Stderr, "Please provide email number to open\n")
			fmt.Fprintf(os.Stderr, "💡 Usage: gotmail open <email-number> [--id <account-id>]\n")
			return 2
		}

		var emailNum int
		if _, err := fmt.Sscanf(positional[0], "%d", &emailNum); err != nil {
			fmt.Fprintf(os.Stderr, "Invalid email number: %v\n", err)
			fmt.Fprintf(os.Stderr, "💡 Usage: gotmail open <email-number> [--id <account-id>]\n")
			return 2
		}

		if hasAccountID {
			// Validate account ID format
			if err := validateAccountID(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid account ID: %v\n", err)
				return 2
			}
			if err := mailManager.OpenEmailByAccountID(accountID, emailNum); err != nil {
				fmt.Fprintf(os.Stderr, "Error opening email: %v\n", err)
				return 1
			}
		} else {
			if err := mailManager.OpenEmail(emailNum); err != nil {
				fmt.Fprintf(os.Stderr, "Error opening email: %v\n", err)
				return 1
			}
		}
		return 0

	case "export":
		if len(positional) < 1 {
			fmt.Fprintf(os.Stderr, "Please provide export folder\n")
			fmt.Fprintf(os.Stderr, "💡 Usage: gotmail export <folder> [--id <account-id>]\n")
			return 2
		}
		exportFolder := positional[0]
		if hasAccountID {
			// Validate account ID format
			if err := validateAccountID(accountID); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid account ID: %v\n", err)
				return 2
			}
			if err := mailManager.ExportAccountByID(accountID, exportFolder); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting account: %v\n", err)
				return 1
			}
		} else {
			// Validate export path
			if err := validateExportPath(exportFolder); err != nil {
				fmt.Fprintf(os.Stderr, "Invalid export path: %v\n", err)
				// 2, not 1: an unusable argument is a usage error, the same as
				// the empty-folder case a few lines above and as the account ID
				// check in the --id branch. 1 is reserved for failures that come
				// from doing the work.
				return 2
			}
			if err := mailManager.ExportAccount(exportFolder); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting account: %v\n", err)
				return 1
			}
		}
		return 0

	case "help":
		if len(positional) > 0 {
			utils.ShowCommandHelp(positional[0])
		} else {
			utils.ShowHelp()
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintf(os.Stderr, "Available commands: new, ls, msg, del, show, open, export, help\n")
		fmt.Fprintf(os.Stderr, "   Use 'gotmail help' for more information\n")
		utils.ShowHelp()
		return 2
	}
}
