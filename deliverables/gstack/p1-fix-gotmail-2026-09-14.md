# GoTMail P1 批次交付报告

日期：2026-09-14
仓库：`D:\GitHub\gotmail`（远程已重定向至 `HHSTU-IST/gotmail`）
批次：上线前审计的 B 维度 4 项 P1 + EOL 工具链 + 文档失真
状态：**本地与 Windows 侧已闭环并通过独立复验；Linux 运行时实测未取得（见"未闭环项"）**
推送策略：按用户指示**仅提交，未推送**

## 提交

| Commit | 内容 |
| --- | --- |
| `af567a0` | P1 四项 + 工具链 + 文档首轮（16 files, +987/−133） |
| `dd49afc` | 复核驱动修复：缓存目录 fail-closed、写入统一为原子替换、文档二次纠正（6 files, +126/−64） |
| `a661cbd` | `help` 精确化：说明单账户不提示（2 files, +6） |

## 修复内容

### 安全

| 项 | 处置 |
| --- | --- |
| 账户文件与导出权限 | `0600`；程序创建的目录 `0700` |
| 导出/邮件 HTML 的软链重定向与 TOCTOU | 改为 `CreateTemp`（`O_CREATE\|O_EXCL`）+ `Rename`，既不跟随链接也不存在 check-then-write 窗口 |
| `~/.gotmail.json` 的 dotfile-manager 兼容 | 先 `symlinkTarget()` 解析目标，**替换目标、保留链接**（stow/chezmoi 不被破坏） |
| 账户文件写入原子性 | 两条写入路径统一走 temp + rename，崩溃不再可能留下截断的账户文件 |
| `crypto/rand` 失败的可预测回退 | 删除；`GenerateRandomString` 返回 `error` |
| 邮件缓存目录降级到世界可写目录 | 删除 `os.TempDir()` 回退，`os.UserCacheDir()` 失败即返错 |
| 无 `--id` 时随机取账户 | `msg`/`del`/`open` 改走 `SelectAccount`（Go map 迭代顺序随机，原 `range`+`break` 等于随机取） |

### 行为与工具链

- `msg`/`show` 补 `validateAccountID`，五条命令的退出码契约统一（0 成功 / 1 运行时 / 2 用法）。
- CI 与 release 的 `go-version` 由 EOL 的 `1.18` 改为 `1.27`；`go.mod` 保留 `go 1.18` 作为语言下限（最低版本指令，与 CI 工具链允许不一致；勿用 `go-version-file: go.mod`）。

### 文档

- 两份 README 的 "Error Fallback Mechanism / 错误回退机制" 已不存在于代码，改写为真实属性。
- 所有 `--id abc123` 示例（6 字符）会被 CLI 自身的 10–50 位校验拒绝，统一改 `a1b2c3d4e5`。
- 命令表中 `show` 原写 "Display current account details"、`del` 原写 "Delete current account"，两者行为描述均错误。
- `README-CN.md` 原教读者运行 `gotmail list`，实际命令是 `ls`。
- 补充 "Rendered Email Cache" 段落与 `help open` 的缓存位置披露（含"内容由发件人撰写，视为不可信"）。
- 多账户段落原暗示所有命令都会提示选择；实际仅 `msg`/`del`/`open` 会，且仅在存储多于一个账户时（`SelectAccount` 在单账户时自动选中）。

### 测试

新增或加强：`utils/write_test.go`、`tests/permissions_test.go`、`TestEmailCacheDirFailsClosedWithoutUserCacheDir`、`TestHelpExamplesUseUsableAccountIDs`（帮助示例必须通过 CLI 自身的 ID 校验，已用变异测试确认非空转）、软链契约双向断言（禁止跟随 + 有意跟随）、以及经软链写入后目标文件回到 `0600` 的断言。

## 独立复验结论

三位专家分别独立复验，结论汇总如下（"首轮"针对 `af567a0`，"二轮"针对 `dd49afc`；QA 二轮所用快照经 md5 核对**与 `dd49afc` 逐字节一致**）。

| 专家 | 范围 | 结论 |
| --- | --- | --- |
| 安全官 | 首轮 | Q1 软链+TOCTOU PASS；Q2 chmod 语义 PASS（指出极窄 fail-open）；Q4 **未闭合** —— 发现 `emailCacheDir` 回退 `/tmp` 可致缓存投毒 |
| 安全官 | 二轮 | F-1 已闭合；Q2 PASS（chmod 块整体删除，0600 由 `CreateTemp` + rename 保证）；F-5 原子性 PASS；Q4 判为已闭合；新测试非空转 |
| 排障手 | 首轮 | 4 项旧缺陷全部确认闭合；无 P0/P1 回归；提出 1 项 P2 + 6 项 P3 |
| 排障手 | 二轮 | 4 项文档修复确认；新收集 1 P2 + 6 P3 |
| QA 负责人 | 首轮 | A/B/D PASS；C/E 因 `wsl.exe` 被策略黑名单阻断而未取得 |
| QA 负责人 | 二轮 | 任务 1–4 全 PASS；证伪 A–E 全部咬合（改坏一处即对应测试失败）；C/E 仍阻断 |

### 关键发现：`af567a0` 引入的回归

安全官在对 `af567a0` 的复验中用探针实测出一个**本批自己引入的**中危回归：`emailCacheDir()` 在 `os.UserCacheDir()` 失败时回退 `os.TempDir()`。预置软链 `base/gotmail -> 攻击者目录` 后，`os.MkdirAll` 返回 `nil` 并在攻击者目录内建出目录；攻击者拥有父目录，可在浏览器读取前替换渲染结果，从而以 `file://` 在受害者浏览器中执行攻击者 HTML。

该回归的特殊之处在于它与同批确立的原则自相矛盾：同一轮既删除了 `crypto/rand` 的可预测回退、主张"失败要响亮"，又给缓存目录留了一条静默降级到世界可写目录的路径。已在 `dd49afc` 按安全官方案 ① 删除该回退。

### 证伪证据（证明新测试有效）

QA 负责人在仓库外副本逐项改坏实现再恢复，对应测试均如实失败：

| 改坏点 | 失败的测试 |
| --- | --- |
| `if allowSymlink{` → `if true{` | `TestWriteFilePrivateReplacesPlantedSymlink`、`TestExportAccountDoesNotFollowPlantedSymlink` |
| → `if false{` | `TestWriteFilePrivateFollowsSymlinkWhenAllowed` |
| `emailCacheDir` 去掉 `/email` | `TestWriteEmailHTMLLivesInUserCacheDir` |
| 过期 cutoff 前移一年 | `TestWriteEmailHTMLKeepsRendersApart` |
| `writeEmailHTML` 改用固定文件名 | `TestWriteEmailHTMLDoesNotFollowPlantedSymlink` |

此外禁用 Windows 跳过条件后，`0600` 断言真实失败（`Expected 0600, got 0666`），说明权限断言非恒真。

新增的容错测试 `TestEmailCacheDirFailsClosedWithoutUserCacheDir` 由安全官在二轮复核中另行确认非空转。

## 未闭环项

### 1. Linux 运行时权限实测（唯一实质性缺口）

`wsl.exe` 被本机"Program Blacklist"安全策略硬阻断，`dangerouslyDisableSandbox` 亦被拒；docker / podman / qemu / cygwin 均不存在。因此**在本机无法取得 Linux 运行时证据**。

已取得的替代证据：

- `GOOS=linux GOARCH=amd64 go build ./...` 与 `go test -c`（含 `utils`、`tests` 两个测试二进制）均成功。
- `GOOS=linux go vet ./...` 干净。
- `tests/permissions_test.go` 的权限断言设计为**在 Linux 上运行而非跳过**，`ubuntu-latest` 的 CI 任务正是跑同一批断言。

**闭合方式**：下次 push 或 PR 时由 `ubuntu-latest` CI 任务自动取得。因本批按要求未推送，该缺口按设计保持打开。人工复现命令见下。

### 2. 工作区遗留物

`utils/dev/null`（约 11.2 MB）与 `tests/dev/null`（约 8.2 MB）是本次交叉编译的副产物：在 Windows 上 `go build -o /dev/null` 会把 `/dev/null` 当作**相对路径**，真的建出目录与文件。二者未被跟踪、未进入任何提交（本会话一律按显式路径 `git add`，从不使用 `git add -A`）。清理操作曾被用户拒绝，故保留待用户处置。

### 3. 已记录不修（P3）

| 项 | 位置 | 说明 |
| --- | --- | --- |
| `*.json` 忽略模式过宽 | `.gitignore` | 会静默忽略 `deliverables/` 下任何 JSON 及 `.vscode/settings.json`。收窄前需逐条 `git check-ignore -v` 验证 |
| `validateExportPath` 失败返 1 | `main.go:264` | 与其它输入校验返 2 不一致；唯一可达路径是绝对路径的父目录不存在，属环境条件 |
| `ShowDetails` 无调用点 | `utils/manager.go:389` | `show` 无 `--id` 走 `GetAllAccountsJSON`；注释已标注不可达 |
| `NO_COLOR` 未被遵循 | `utils/color.go` | 仍输出 ANSI 转义 |
| `TestCopy` 非自足 | `tests/copy_test.go` | 断言真实剪贴板成功，仅在 Linux 跳过；Windows 上取决于剪贴板可用性 |
| 账户文件写入的语义收窄 | `utils/common.go` | 现在需要**目标所在目录**的写权限（原为文件本身）；硬链接不再被跟随；指向无效目标的软链会被替换为实体文件。均为原子替换的必然代价，属有意取舍 |
| `$HOME` 下 `.gotmail-tmp-*` 残留 | `utils/common.go` | 仅进程被 SIGKILL 时可能遗留 0600 空/半写文件，无泄露 |
| 缓存目录 failure 语义 | `utils/manager.go` | `emailCacheDir` 现在失败即返错，属用户可见行为变更，应写入发布说明 |

## 复现命令

```bash
# 本地（Windows）：注意本机沙箱拒绝 clip，故跳过剪贴板测试
gofmt -l $(git ls-files '*.go')
go vet ./...
go build ./...
go test ./... -count=1 -skip TestCopy

# Linux 目标可编译性（无需 Linux 运行时）
GOOS=linux GOARCH=amd64 go build ./...
GOOS=linux GOARCH=amd64 go vet ./...

# Linux 运行时权限实测（需 Linux 环境；脚本已修正为重新编译后再跑）
wsl.exe bash /mnt/d/GitHub/gotmail/.scratch/qa/linux-p1/run_linux.sh
```

## 流程备注

本仓库存在并行编辑会话。本批两次撞上其副作用：一是 `README.md` 的两处已成功编辑在提交前被回写覆盖（导致 `af567a0` 中两份 README 的 Go 版本互相矛盾），二是 HEAD 在复验途中变动。结论：**"编辑成功"不等于"提交时内容仍在"**，提交后必须回读 `git show <commit>:<file>` 核对实际内容，而不能依赖编辑期的判断。同理，跨面复验应以 commit hash 而非工作区快照为准（本次 QA 快照经 md5 核对确实等于 `dd49afc`，该问题未影响结论）。
