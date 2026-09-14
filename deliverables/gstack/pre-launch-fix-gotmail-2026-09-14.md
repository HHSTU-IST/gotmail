# GoTMail 上线前全检 · P0 修复与独立复验报告

- 日期：2026-09-14
- 场景：上线前检查（代码审查 + 安全审计 + QA 测试）→ 阻断项修复 + 独立复验
- 基准：`main` @ `109b4c0`（全检）→ `4091b9d`（修复后，已推送 `origin/main`）
- 参与成员：主理人（实现） + 质量门神 gstack-qa-lead（QA 复验） + 排障手 gstack-investigator（回归复核）
- 输入：`.scratch/qa/pre-release-audit-2026-09-14.md`（全检报告，21 项发现 / 4 项 P0）

## 📌 TL;DR

- **结论：🟢 可放行。** 4 项 P0 阻断全部修复，并经独立复验通过，含基线 A/B 铁证对照。
- **加权得分 57.25 → 83.25**，四个维度均已消除 P0。
- 新增 CI 门禁在修复提交的 push 上**真实跑通**：7 步 19 秒全绿。
- 4 项 P1（明文凭据 0644、随机回退确定性、账号选择随机化、`open` 落盘位置）与 Go 1.18 EOL 工具链**未纳入本轮**，转入下一批。
- 复验中发现并当场修复 2 条 P3（`show` / `msg` 退出码一致性、`help -- open` 主题错位）；补丁尚在工作区未提交。
- 过程事件：`docs/agents/` 三份未跟踪文档被误删，已从种子模板恢复并入该提交。

## 🎯 核心结论卡片

| 项目 | 内容 |
| ---- | ---- |
| Go / No-Go | 🟢 Go |
| 阻断项 | 4 项 → **0 项** |
| 严重度分布 | P0 4→0 · P1 4→4 · P2 6→6 · P3 7→7 |
| 加权得分 | 57.25 → **83.25** |
| 关键行动项 | 5 条（见行动清单） |
| 需外部拍板 | 2 条：Go 版本选型、`.vscode/` 与 `deliverables/` 的提交策略 |

评分模型（可复算）：每维度起 100 分，P0 扣 25 / P1 扣 10 / P2 扣 4 / P3 扣 1；总分 = 四维度算术平均；任一维度出现 P0 即 NO-GO。

| 维度 | 修复前 | 修复后 | 变化依据 |
| ---- | ------ | ------ | -------- |
| A 代码审查 | 59 | 84 | A-1（P0）修复 |
| B 安全审计 | 65 | 65 | 本轮未动 |
| C QA 与测试 | 31 | 85 | C-1 / C-2（P0）+ C-4（P2）修复 |
| D 仓库与发布 | 74 | 99 | D-1（P0）修复 |
| **合计** | **57.25** | **83.25** | |

## 🚦 放行决策

**阻塞项清单：空。** 4 项 P0 全部关闭，且每一项都有修复前后的实测对照。

**回滚预案。** 改动集中在入口层（`main.go` 的参数解析与退出码）、测试与 CI 配置，**不涉及数据格式变更、不涉及迁移**。回滚方式：`git revert 4091b9d`，即恢复为 `109b4c0` 的行为；用户侧的 `~/.gotmail.json` 无需任何处理。唯一需要留意的是回滚会同时撤销 `.gitignore` 的 `/agents/` 修复，`docs/agents/` 会重新被忽略。

## 1. 各成员核心结论

### ✅ 质量门神（gstack-qa-lead）· QA 测试

- 核心判断：**CONDITIONAL PASS**，四项 P0 全部独立复验通过。
- 关键证据：用 `git archive 109b4c0` 重建基线二进制做 A/B。基线 `export <dir> --id zz` 为 **exit 0 且写出全部账号**；`badcommand` / 缺参均 **exit 0**。修复后分别为 exit 1 / exit 2 —— 证明是**真实修复而非空操作**。
- 关键纠正：把空值 `--id` 从 P1 降为 **P2**。`del --id=` 与裸 `del` 逐字等价，走 `SelectAccount` 交互提示；非交互 stdin 下 exit 1、**未删任何账号**。最坏情形等于用户自己敲 `del`。
- **delta 复验：PASS**（在 `4091b9d` 上重跑）。空值 `--id=` 在所有命令上 exit 2，含最危险的单账号 `del --id=`（原先会删掉唯一账号）；`--` 终止符在 `export -- <dir>` / `export <dir> --` / `export <dir> -- --id ...` / `open -- 1` 四种写法下语义正确；回归矩阵全绿。
- 新增两条 P3，**均由主理人当场修复**：`show` / `msg` 未调 `validateAccountID`，导致"用法错误 = 2"只覆盖 3/5 命令；`help -- open` 的主题错位到 `--`。
- 隔离手段：`USERPROFILE` 与 `HOME` 同指假 HOME（双账号 `home_d2` / 单账号 `home_d1`）；真实 `~/.gotmail.json` 前后均不存在（未污染）。

### 🔧 排障手（gstack-investigator）· 代码审查 / 回归复核

- 核心判断：**除 1 处外未发现行为回归**；索引平移逐条等价、全仓仅 `main.go` 一处 `os.Exit`、退出码自洽且无隐式落 0、`splitArgs` 无 panic。
- 抓到的真回归：**`--` 终止符失效**（旧 `flag` 包原生支持，新 `splitArgs` 未复刻）—— 已修。
- 主动更正：撤回自己先前对空值 `--id` 的 P1 升级，接受 P2 定级，并自证其"静默删默认账号"的根因假设有误。
- 新增发现：CI/发布工具链钉在 Go 1.18（EOL）为 P2（既有）。其初版归因（称由 `4091b9d` 下调）有误，经逐提交读 diff 后**自行更正**为 `366bb40`，与主理人独立取证一致。
- **delta 代码轴复核：PASS**（实跑二进制，非静态阅读）。`open -- 3` → exit 1 且进入真实路径（修复前为 exit 2 报未知旗标）；`open -- --id abcdefghij` → exit 2，与旧 `flag` 语义一致；`open --id abcdefghij -- 3` → accountID 保留、位置参数取 3；空值四条（`open --id=` / `open 3 --id=` / `del --id=` / `show --id=`）全 exit 2；`open 3 --id abcdefghij` 未回归；`gofmt` / `vet` / `test` / `build` 全绿。
- 附带确认：把 CI 的格式化检查收窄为 `gofmt -l $(git ls-files '*.go')` 是必要的——本地 `gofmt -l .` 会列出 13 个 `.scratch/qa/baseline/*.go`（基线副本），旧写法在开发者本机会假阳性。

### 🧭 主理人（team-lead）· 实现

- 落盘 4 项 P0 + 3 项顺带修复；对 delta 自行补做了一遍 E2E 退出码验证（QA 的 delta 回执未在时限内返回，记为局限）。

## 2. 综合发现（去重合并，按严重度）

| # | 严重度 | 类别 | 位置 | 问题 | 处置 | 来源 |
|---|--------|------|------|------|------|------|
| 1 | 🔴 P0 | 仓库 | `.gitignore:35` | 裸模式 `agents` 吞掉 `docs/agents/`，三份技能配置永不入库 | 已修 → `/agents/` | 主理人 |
| 2 | 🔴 P0 | CI | `release.yml:3-29` | 无 PR/分支门禁；测试命令漏掉根包与 `utils` | 已修 → 新增 `ci.yml`，命令改 `go vet ./... && go test ./... -count=1` | 主理人 |
| 3 | 🔴 P0 | QA | `main_test.go:12-25` | `captureOutput` 只重定向 stdout，根包测试恒红 | 已修 → 双流捕获，测试改调 `run()` | 主理人 |
| 4 | 🔴 P0 | 代码 | `main.go:83` | 位置参数之后的 `--id` 被静默丢弃 → **操作错误账号** | 已修 → 新增 `splitArgs`，位置无关 | 主理人 |
| 5 | 🟠 P2 | 代码 | `splitArgs` | 显式空 `--id=` 静默降级为"未指定 id" | 已修 → 取空值报错，exit 2 | 排障手 / 门神 |
| 6 | 🟠 P2 | 代码 | `main.go` 各命令 | 错误路径退出码恒为 0，脚本无法判成败 | 已修 → 0 成功 / 1 运行时 / 2 用法 | 主理人 |
| 7 | 🟠 P2 | 回归 | `main.go:68` | `--` 终止符不再被识别（本轮修复引入的唯一回归） | 已修 → `--` 后按位置参数并停止解析 | 排障手 |
| 8 | 🟠 P2 | CI | `ci.yml` / `release.yml` | 工具链钉在 Go 1.18（EOL），发布产物无安全补丁 | **未修**，转下一批 | 排障手 |
| 9 | 🟡 P3 | 文档 | `README.md:158` / `README-CN.md:158` | 教的测试命令漏掉根包 | 已修 → `go test ./... -v` | 排障手 |
| 10 | 🟡 P3 | CI | 两个 workflow | 无 `go.sum` 却设 `cache: true`（仅告警、不致命） | 已修 → 删除 | 门神 |
| 11 | 🟡 P3 | 仓库 | `.gitignore` | `.vscode/` 被移除（非本轮改动） | **未修**，待确认 | 门神 |
| 12 | 🟡 P3 | 代码 | `main.go:264` | `validateExportPath` 失败返 1，与其它输入校验返 2 不一致 | **未修**（唯一可达路径是绝对路径的父目录不存在，属环境条件，返 1 可自圆） | 排障手 |
| 13 | 🟡 P3 | 代码 | `main.go` `msg` / `show` 分支 | 未调 `validateAccountID`，"用法错误 = 2"只覆盖 3/5 命令 | 已修 → 两处补校验，五条命令统一 exit 2 | 门神 |
| 14 | 🟡 P3 | 代码 | `main.go` `help` 分支 | 主题读的是原始 argv 而非解析后的位置参数，`help -- open` 主题错位到 `--` | 已修 → 改用 `positional[0]` | 门神 |
| 15 | ⚪ 说明 | 行为 | `main.go` | 未知/多余旗标由"静默忽略"变为 exit 2 | 有意收紧，进 release notes | 门神 |

## 🔁 本轮有意变更的行为（建议进 release notes）

> 退出码语义收紧（用法错误 2 / 运行时错误 1，原先错误路径一律 0）；未知或多余旗标不再静默忽略；`gotmail open 3 --id abc123` 这类"旗标写在位置参数之后"的写法，由**静默失效**变为**正确生效**；显式空 `--id=` 不再降级为"未指定"。

## ✅ 行动清单

| # | 行动 | 负责方 | 紧急度 |
|---|------|--------|--------|
| 1 | **B-1**：`~/.gotmail.json` 与导出文件写权限 `0644` → `0600`（含明文密码与 JWT） | 维护者 | P1 |
| 2 | **B-2**：`GenerateRandomString` 的 `crypto/rand` 回退改为报错，禁止静默降级为确定性序列 | 维护者 | P1 |
| 3 | **A-2 / C-3**：不指定 `--id` 时的账号选择改用 `SelectAccount`；`open` 的 `email.html` 落盘改到用户缓存/临时目录 | 维护者 | P1 |
| 4 | 决定 CI/发布的 Go 版本（建议 `go-version: "1.23"`；**不要**用 `go-version-file: go.mod`，它读到 1.18，起不到作用） | 维护者 | P2 |
| 5 | 确认 `.vscode/` 是否恢复进 `.gitignore`；确认 `deliverables/`、`AGENTS.md`、`docs/agents/` 的提交策略 | 维护者 | P3 |

## ⚠️ 待完善 / 已知局限

- **delta 由三方交叉覆盖**：排障手做代码轴复核（实跑二进制，PASS）；QA 做 E2E 轴复核（PASS，含单账号 `del --id=` 这一最危险场景）；主理人做退出码 E2E。三方结论一致。
- **E4/E5 两条 P3 的补丁尚未提交**：`main.go` 的 `msg` / `show` 校验与 `help` 主题索引修复落在 `4091b9d` 之后，属未提交的工作区改动，需另行提交。
- **未做真实 API 端到端联调**（需 mail.tm 配额）；`createAccountAPI` / `getToken` 的成功路径仅静态审查。
- **未做三平台实机验证**；`0600` 权限位、`open` 落盘路径、Linux 剪贴板选区等结论来自平台语义分析，不是本机实测。
- **B 维度本轮未动**，四项 P1 仍在，安全得分维持 65。
- **过程事件一**：`docs/agents/` 三份未跟踪文档在 09:11 被误删。已从 `~/.workbuddy/skills/setup-matt-pocock-skills/` 种子模板恢复并随提交入库。根因未完全定位——已向 QA 追问其复现脚本是否含 `git clean` / `git stash -u` / `rm`，回执未到。
- **过程事件二**：`4091b9d` 的提交与推送由用户的并行编辑会话完成，非本团队操作。已核对提交内容与本地验证态逐字节一致。

## 📚 成员产出索引

- **gstack-qa-lead（质量门神）**：`.scratch/qa/verify-report-2026-09-14.md`（含原始 stdout、exit code、基线 A/B、复现脚本 `.scratch/qa/verify/harness.sh`、日志 `finding1*.log`）
- **gstack-investigator（排障手）**：无独立落盘，结论以消息回传；其 Bash 输出通道异常，证据经重定向到文件后读取，复现记录在 `D:\tmp\probe\*.txt`
- **主理人（team-lead）**：实现落盘于 `main.go` / `main_test.go` / `.github/workflows/ci.yml` / `.github/workflows/release.yml` / `.gitignore` / `README.md` / `README-CN.md`；全检原始报告见 `.scratch/qa/pre-release-audit-2026-09-14.md`

## 变更清单（修复后 `4091b9d`）

| 文件 | 改动 |
|------|------|
| `main.go` | 重构为 `run(args []string) int`；新增 `splitArgs`（位置无关的 `--id`、`--` 终止符、空值报错、未知旗标拒绝）；退出码 0/1/2；`msg` / `show` 补 `validateAccountID`；`help` 主题改读 `positional[0]` |
| `main_test.go` | `captureOutput` 捕获双流；测试改调 `run()`；`TestSplitArgs` 14 个子用例；`TestRunWithMalformedAccountID` 5 命令表驱动；`TestRunHelpResolvesTopicFromPositional` 2 例 |
| `.github/workflows/ci.yml` | 新增：push / pull_request 门禁（gofmt / vet / build / test） |
| `.github/workflows/release.yml` | 测试命令改 `go vet ./... && go test ./... -count=1`；移除 `cache: true` |
| `.gitignore` | `agents` → `/agents/` |
| `README.md` / `README-CN.md` | 测试命令改 `go test ./... -v` |
| `AGENTS.md` / `docs/agents/*` | 上一轮 setup 交付物，随本次提交入库 |

> 上表对应已提交的 `4091b9d`。`main.go` / `main_test.go` 的 E4/E5 补丁（`msg` / `show` 补校验 + `help` 主题索引）在其之后，**尚在工作区、未提交**。

> 本报告由软件工坊 AI 协作生成，关键决策请由工程负责人复核。
