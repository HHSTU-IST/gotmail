# GoTMail

[![Go Version](https://img.shields.io/badge/go-1.18+-blue.svg)](https://golang.org/doc/go1.18)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivaquero/gotmail)](https://goreportcard.com/report/github.com/ivaquero/gotmail)
![code size](https://img.shields.io/github/languages/code-size/ivaquero/gotmail.svg)
![repo size](https://img.shields.io/github/repo-size/ivaquero/gotmail.svg)

GoTMail (Go Temporary Mail) is temporary email CLI tool written in Go for [mail.tm](https://mail.tm/), providing cross-platform temporary email management functionality.

**[中文版本](README-CN.md)**

## 🌟 Features

- **Multi-Account Management**: Support creating and managing multiple temporary email accounts (up to 10)
- **Temporary Email Creation**: Quickly create temporary email accounts
- **Message Management**: Fetch and list received email messages
- **Email Viewing**: Open specific emails in your browser
- **Account Management**: View account details and delete accounts
- **Data Export**: Export all accounts or specific account data to specified paths
- **Cross-platform Support**: Support for Windows, macOS, and Linux
- **Clipboard Integration**: Automatically copy email addresses to clipboard
- **Command Line Interface**: Simple and easy-to-use CLI operations
- **Backward Compatibility**: Support automatic conversion of old single-account files

## 🚀 Quick Start

### Installation

- Windows

```bash
scoop bucket add scoopforge/main-plus
scoop install gotmail
```

- macOS and Linux

```bash
brew tap brewforge/more
brew install gotmail
```

> After installation, run `xattr -r -d com.apple.quarantine $HOMEBREW_PREFIX/bin/gotmail` to allow execution.

- Alternative method (requires Go 1.18 or higher)

```bash
go install github.com/ivaquero/gotmail
```

### Basic Usage

#### Account Management

Create a new temporary email account:

```bash
gotmail new
```

List all accounts:

```bash
gotmail ls
```

View account details:

```bash
gotmail show
```

Delete account:

```bash
gotmail del
```

#### Message Management

View received emails:

```bash
gotmail msg
```

Open a specific email in your browser:

```bash
gotmail open 1
```

#### Data Management

Export account data:

```bash
gotmail export /backup/folder/
```

#### Multi-Account Operations

For multi-account scenarios, most commands support the `--id` parameter to specify a particular account:

```bash
# View emails for a specific account
gotmail msg --id a1b2c3d4e5

# View details for a specific account
gotmail show --id a1b2c3d4e5

# Delete a specific account
gotmail del --id a1b2c3d4e5

# Open specific email from specific account
gotmail open 1 --id a1b2c3d4e5

# Export specific account data
gotmail export ./backup/folder --id a1b2c3d4e5
```

## 📖 Command Reference

| Command                     | Description                                      | Example                                    |
| --------------------------- | ------------------------------------------------ | ------------------------------------------ |
| `new`                       | Create a new temporary email account             | `gotmail new`                              |
| `ls`                        | List all accounts                                | `gotmail ls`                               |
| `msg`                       | Fetch and list all emails                        | `gotmail msg`                              |
| `msg --id <id>`             | Fetch emails for a specific account              | `gotmail msg --id a1b2c3d4e5`                  |
| `open <number>`             | Open specified email in browser                  | `gotmail open 1`                           |
| `open <number> --id <id>`   | Open specified email for specific account        | `gotmail open 1 --id a1b2c3d4e5`               |
| `show`                      | Display current account details                  | `gotmail show`                             |
| `show --id <id>`            | Display specific account details                 | `gotmail show --id a1b2c3d4e5`                 |
| `del`                       | Delete current account                           | `gotmail del`                              |
| `del --id <id>`             | Delete specific account                          | `gotmail del --id a1b2c3d4e5`                  |
| `export <folder>`           | Export all account data to specified folder      | `gotmail export backup/folder`             |
| `export <folder> --id <id>` | Export specific account data to specified folder | `gotmail export backup/folder --id a1b2c3d4e5` |
| `help`                      | Show help information                            | `gotmail help`                             |
| `help <command>`            | Show detailed help for specific command          | `gotmail help msg`                         |

## 🔧 Development Guide

### Requirements

- Go 1.27 or higher (`go.mod` still declares `go 1.18` as the language floor)

### Building the Project

```bash
git clone https://github.com/ivaquero/gotmail
cd gotmail
go build
```

### Running Tests

```bash
go test ./... -v
```

### Code Standards

- Follow Go standard code formatting
- Use structured error handling
- Provide detailed error information
- Maintain cross-platform compatibility

## 🔒 Security Features

- **Cryptographic Random Generation**: Use `crypto/rand` to generate secure random strings
- **Fail-Closed Random Generation**: Abort with an error instead of falling back to a predictable sequence when `crypto/rand` fails
- **Owner-Only File Permissions**: Account data and exports are written `0600`; directories the program creates are `0700`
- **Symlink-Safe Writes**: Exports and cached email HTML are written to a fresh file and renamed into place, so a symbolic link planted at the destination cannot redirect the write
- **Input Validation**: Validate API responses and user inputs
- **Secure Data Storage**: Store account data securely in JSON format

## 🌐 API Integration

This project uses the Mail.tm API to provide temporary email services:

- **API Endpoint**: `https://api.mail.tm`
- **Feature Support**: Account creation, email retrieval, account deletion
- **Data Format**: JSON
- **Authentication**: Bearer Token

## 📝 Data Storage

Account data is stored in a local file:

- **File Path**: `~/.gotmail.json`
- **Data Format**: JSON
- **Contains**: Multiple accounts with ID, email address, password, authentication token
- **Backward Compatibility**: Automatically converts old single-account file formats

### Data Export

You can export all account data to any specified path using the `export` command:

```bash
gotmail export /path/to/backup/
```

Or export data for a specific account:

```bash
gotmail export /path/to/backup/ --id a1b2c3d4e5
```

The exported file will be an exact copy of the original account data file, preserving all account information and formatting.

### Rendered Email Cache

`gotmail open` hands the message to your browser as a file rather than piping it, so each render is written to the user cache directory:

- **Directory**: `<user cache dir>/gotmail/email` — `~/.cache/gotmail/email` on Linux, `~/Library/Caches/gotmail/email` on macOS, `%LocalAppData%\gotmail\email` on Windows
- **Permissions**: `0600`, and every render gets its own file so a second `open` cannot overwrite what the browser is still displaying
- **Retention**: renders older than one hour are removed the next time you run `open`. The file is not deleted right after the browser is launched, because that launch is asynchronous and deleting it would race the viewer
- **Caution**: the HTML is authored by whoever sent the email. Treat the cached file as untrusted content, and delete it yourself if you want it gone before the next cleanup

### Multi-Account Management

GoTMail now supports creating and managing multiple temporary email accounts (up to 10):

1. **Create New Account**: Use `gotmail new` to create a new account
2. **View All Accounts**: Use `gotmail ls` to list all created accounts
3. **Account-Specific Operations**: Most commands support the `--id <account_id>` parameter to specify which account to operate on
4. **Backward Compatibility**: For single-account scenarios, commands can still be used without the `--id` parameter

When performing operations that require an account, if multiple accounts exist, the system will prompt you to select which account to use.

## 🐛 Error Handling

The project implements comprehensive error handling mechanisms:

- **Network Errors**: Handle API connection failures
- **File Operation Errors**: Handle data read/write failures
- **Clipboard Errors**: Handle cross-platform clipboard operation failures
- **API Response Errors**: Handle API error status returns

## 🤝 Contributing Guidelines

Issues and Pull Requests are welcome to improve the project:

1. Fork the project repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add some amazing feature'`)
4. Push to branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## 📄 License

This project is licensed under the MIT License.

## 🙏 Acknowledgments

- [mail.tm](https://mail.tm) for providing temporary email services

---

**Note**: This is a temporary email tool, please do not use it to receive important or sensitive information.
