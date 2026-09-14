package utils

import (
	"fmt"
)

// ShowHelp displays general help information
func ShowHelp() {
	fmt.Println("Your Temporary Email Accounts Manager")
	fmt.Println("\nUsage:")
	fmt.Println("  new                           Create a new account")
	fmt.Println("  ls                            List all accounts")
	fmt.Println("  msg [--id <id>]               Fetch and list messages")
	fmt.Println("  del [--id <id>]               Delete account")
	fmt.Println("  show [--id <id>]              Show account details or all accounts in JSON")
	fmt.Println("  open <number> [--id <id>]     Open specific email in browser")
	fmt.Println("  export <folder> [--id <id>]   Export account data to specified folder")
	fmt.Println("  help                          Show this help message")
	fmt.Println("\nOptions:")
	fmt.Println("  --id <id>                     Specify account ID (10-50 letters and digits)")
}

// ShowCommandHelp displays detailed help for specific command
func ShowCommandHelp(command string) {
	switch command {
	case "new":
		fmt.Println("Create a new account")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail new")
		fmt.Println("\nDescription:")
		fmt.Println("  Creates a new temporary email account with a random address")
		fmt.Println("  The account credentials will be stored locally for future use")

	case "ls":
		fmt.Println("List all accounts")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail ls")
		fmt.Println("\nDescription:")
		fmt.Println("  Displays all stored email accounts with their IDs and addresses")

	case "msg":
		fmt.Println("Fetch and list messages")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail msg [--id <account_id>]")
		fmt.Println("\nDescription:")
		fmt.Println("  Retrieves messages from the specified account")
		fmt.Println("  When --id is omitted, prompts you to pick one of the stored accounts")
		fmt.Println("\nExamples:")
		fmt.Println("  gotmail msg                    # Pick an account interactively")
		fmt.Println("  gotmail msg --id a1b2c3d4e5    # Fetch from a specific account")

	case "del":
		fmt.Println("Delete account")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail del [--id <account_id>]")
		fmt.Println("\nDescription:")
		fmt.Println("  Removes the specified account from storage")
		fmt.Println("  When --id is omitted, prompts you to pick one of the stored accounts")
		fmt.Println("\nExamples:")
		fmt.Println("  gotmail del                    # Pick an account interactively")
		fmt.Println("  gotmail del --id a1b2c3d4e5    # Delete a specific account")

	case "show":
		fmt.Println("Show account details or all accounts")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail show [--id <account_id>]")
		fmt.Println("\nDescription:")
		fmt.Println("  When --id is specified, shows detailed information about the specific account")
		fmt.Println("  When --id is not specified, shows all accounts data in JSON format")
		fmt.Println("\nExamples:")
		fmt.Println("  gotmail show                   # Show all accounts in JSON format")
		fmt.Println("  gotmail show --id a1b2c3d4e5   # Show specific account details")

	case "open":
		fmt.Println("Open specific email in browser")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail open <number> [--id <account_id>]")
		fmt.Println("\nDescription:")
		fmt.Println("  Opens the specified email message in your default web browser")
		fmt.Println("  When --id is omitted, prompts you to pick one of the stored accounts")
		fmt.Println("  The message is rendered to an HTML file under the user cache directory")
		fmt.Println("  (<cache>/gotmail/email). Renders older than an hour are removed on the")
		fmt.Println("  next open, but the file is not deleted while the browser may still be")
		fmt.Println("  reading it. The HTML comes from the sender, so treat it as untrusted.")
		fmt.Println("\nExamples:")
		fmt.Println("  gotmail open 1                 # Pick an account, open its first message")
		fmt.Println("  gotmail open 3 --id a1b2c3d4e5 # Open message 3 of a specific account")

	case "export":
		fmt.Println("Export account data to specified folder")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail export <folder> [--id <account_id>]")
		fmt.Println("\nDescription:")
		fmt.Println("  Exports account data to the specified folder")
		fmt.Println("  When --id is omitted, exports every stored account")
		fmt.Println("  The export holds plaintext passwords and tokens; it is written")
		fmt.Println("  owner-only (0600), and a directory this command creates is 0700")
		fmt.Println("\nExamples:")
		fmt.Println("  gotmail export /tmp/backup                 # Export all accounts")
		fmt.Println("  gotmail export /tmp/backup --id a1b2c3d4e5 # Export one account")

	case "help":
		fmt.Println("Show help information")
		fmt.Println("\nUsage:")
		fmt.Println("  gotmail help [command]")
		fmt.Println("\nDescription:")
		fmt.Println("  Shows general help or detailed help for a specific command")
		fmt.Println("\nExamples:")
		fmt.Println("  gotmail help                   # Show general help")
		fmt.Println("  gotmail help msg               # Show help for msg command")

	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("\nAvailable commands:")
		fmt.Println("  new, ls, msg, del, show, open, export, help")
		fmt.Println("\nUse 'gotmail help <command>' for detailed help on a specific command")
	}
}
