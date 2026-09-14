# GoTMail

[![Go Version](https://img.shields.io/badge/go-1.27+-blue.svg)](https://golang.org/doc/go1.27)
[![Go Report Card](https://goreportcard.com/badge/github.com/ivaquero/gotmail)](https://goreportcard.com/report/github.com/ivaquero/gotmail)
![code size](https://img.shields.io/github/languages/code-size/ivaquero/gotmail.svg)
![repo size](https://img.shields.io/github/repo-size/ivaquero/gotmail.svg)

GoTMail（Go Temporary Mail）是一个用 Go 语言编写的 [mail.tm](https://mail.tm/) 临时邮箱 CLI 工具，提供跨平台的临时邮箱管理功能。

**[English](README.md)**

## 🌟 功能特性

- **多账户管理**：支持创建和管理多个临时邮箱账户（最多10个）
- **临时邮箱创建**：快速创建临时邮箱账户
- **消息管理**：获取和列出收到的邮件消息
- **邮件查看**：在浏览器中打开特定邮件
- **账户管理**：查看账户详情和删除账户
- **数据导出**：导出所有账户或指定账户数据到指定路径
- **跨平台支持**：支持 Windows、macOS 和 Linux
- **剪贴板集成**：自动复制邮箱地址到剪贴板
- **命令行界面**：简单易用的 CLI 操作
- **向后兼容**：支持旧版单账户文件的自动转换

## 🚀 快速开始

### 安装

- Windows

```bash
scoop bucket add scoopforge/main-plus
scoop install gotmail
```

- macOS 和 Linux

```bash
brew tap brewforge/more
brew install gotmail
```

> 安装完成后，运行 `xattr -r -d com.apple.quarantine $HOMEBREW_PREFIX/bin/gotmail` 以允许执行。

- 备选方法（需要 Go 1.27 或更高版本，即 CI 构建所用的工具链）

```bash
go install github.com/ivaquero/gotmail
```

### 基本用法

#### 账户管理

创建新的临时邮箱账户：

```bash
gotmail new
```

列出所有账户：

```bash
gotmail ls
```

查看账户信息：

```bash
gotmail show
```

删除账户：

```bash
gotmail del
```

#### 消息管理

查看收到的邮件：

```bash
gotmail msg
```

在浏览器中打开特定邮件：

```bash
gotmail open 1
```

#### 数据管理

导出账户数据：

```bash
gotmail export /备份文件夹/
```

#### 多账户操作

对于多账户场景，大多数命令支持 `--id` 参数来指定特定账户：

```bash
# 查看指定账户的邮件
gotmail msg --id a1b2c3d4e5

# 查看指定账户信息
gotmail show --id a1b2c3d4e5

# 删除指定账户
gotmail del --id a1b2c3d4e5

# 从指定账户打开特定邮件
gotmail open 1 --id a1b2c3d4e5

# 导出指定账户数据
gotmail export ./备份文件夹 --id a1b2c3d4e5
```

## 📖 命令参考

| 命令                        | 描述                         | 示例                                       |
| --------------------------- | ---------------------------- | ------------------------------------------ |
| `new`                       | 创建新的临时邮箱账户         | `gotmail new`                              |
| `ls`                        | 列出所有账户                 | `gotmail ls`                               |
| `msg`                       | 获取并列出所有邮件           | `gotmail msg`                              |
| `msg --id <id>`             | 获取指定账户的邮件           | `gotmail msg --id a1b2c3d4e5`                  |
| `open <number>`             | 在浏览器中打开指定邮件       | `gotmail open 1`                           |
| `open <number> --id <id>`   | 为指定账户打开指定邮件       | `gotmail open 1 --id a1b2c3d4e5`               |
| `show`                      | 显示当前账户详情             | `gotmail show`                             |
| `show --id <id>`            | 显示指定账户详情             | `gotmail show --id a1b2c3d4e5`                 |
| `del`                       | 删除当前账户                 | `gotmail del`                              |
| `del --id <id>`             | 删除指定账户                 | `gotmail del --id a1b2c3d4e5`                  |
| `export <folder>`           | 导出所有账户数据到指定文件夹 | `gotmail export backup/folder`             |
| `export <folder> --id <id>` | 导出指定账户数据到指定文件夹 | `gotmail export backup/folder --id a1b2c3d4e5` |
| `help`                      | 显示帮助信息                 | `gotmail help`                             |
| `help <command>`            | 显示特定命令的详细帮助       | `gotmail help msg`                         |

## 🔧 开发指南

### 环境要求

- Go 1.27 或更高版本（`go.mod` 仍声明 `go 1.18` 作为语言下限）

### 构建项目

```bash
git clone https://github.com/ivaquero/gotmail
cd gotmail
go build
```

### 运行测试

```bash
go test ./... -v
```

### 代码规范

- 遵循 Go 标准代码格式
- 使用结构化的错误处理
- 提供详细的错误信息
- 保持跨平台兼容性

## 🔒 安全特性

- **加密随机数生成**：使用 `crypto/rand` 生成安全的随机字符串
- **失败即中止的随机数生成**：`crypto/rand` 失败时直接报错，不回退到可预测的序列
- **仅属主可读的文件权限**：账户数据与导出文件写入权限为 `0600`，程序创建的目录为 `0700`
- **防符号链接写入**：导出文件与缓存的邮件 HTML 先写入新文件再重命名就位，攻击者预置的符号链接无法把写入重定向到其他文件
- **输入验证**：对 API 响应和用户输入进行验证
- **安全的数据存储**：账户数据以 JSON 格式安全存储

## 🌐 API 集成

本项目使用 Mail.tm API 提供临时邮箱服务：

- **API 端点**: `https://api.mail.tm`
- **功能支持**: 账户创建、邮件获取、账户删除
- **数据格式**: JSON
- **认证方式**: Bearer Token

## 📝 数据存储

账户数据存储在本地文件中：

- **文件路径**: `~/.gotmail.json`
- **数据格式**: JSON
- **包含信息**: 多个账户的 ID、邮箱地址、密码、认证令牌
- **向后兼容**: 自动转换旧的单账户文件格式

### 数据导出

您可以使用 `export` 命令将所有账户数据导出到任意指定路径：

```bash
gotmail export /path/to/backup/
```

或者导出指定账户的数据：

```bash
gotmail export /path/to/backup/ --id a1b2c3d4e5
```

导出的文件将是原始账户数据文件的完整副本，保留所有账户信息和格式。

### 邮件渲染缓存

`gotmail open` 会把邮件作为文件交给浏览器打开，而不是通过管道传给浏览器，因此每次渲染都会写入用户缓存目录：

- **目录**：`<用户缓存目录>/gotmail/email` —— Linux 下为 `~/.cache/gotmail/email`，macOS 下为 `~/Library/Caches/gotmail/email`，Windows 下为 `%LocalAppData%\gotmail\email`
- **权限**：`0600`，且每次渲染都生成独立文件，因此第二次 `open` 不会覆盖浏览器仍在显示的内容
- **保留时间**：超过一小时的渲染文件会在下次执行 `open` 时删除。文件不会在启动浏览器后立即删除，因为该启动是异步的，立即删除会与浏览器争用
- **注意**：该 HTML 由发件人撰写。请把缓存文件视为不可信内容；若希望它在下次清理前就消失，请自行删除

### 多账户管理

GoTMail 现在支持创建和管理多个临时邮箱账户（最多10个）：

1. **创建新账户**：使用 `gotmail new` 创建新账户
2. **查看所有账户**：使用 `gotmail list` 列出所有已创建的账户
3. **账户特定操作**：大多数命令支持 `--id <account_id>` 参数来指定要操作的账户
4. **向后兼容**：对于只有一个账户的情况，命令仍然可以不带 `--id` 参数使用

当执行需要账户的操作时，如果存在多个账户，系统会提示您选择要使用的账户。

## 🐛 错误处理

项目实现了完善的错误处理机制：

- **网络错误**：处理 API 连接失败
- **文件操作错误**：处理数据读写失败
- **剪贴板错误**：处理跨平台剪贴板操作失败
- **API 响应错误**：处理 API 返回的错误状态

## 🤝 贡献指南

欢迎提交 Issue 和 Pull Request 来改进项目：

1. Fork 项目仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 📄 许可证

本项目采用 MIT 许可。

## 🙏 致谢

- [mail.tm](https://mail.tm) 提供临时邮箱服务

---

**注意**: 这是一个临时邮箱工具，请勿用于接收重要或敏感信息。
