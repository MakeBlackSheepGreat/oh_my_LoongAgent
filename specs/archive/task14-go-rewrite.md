# task14：Go 后端重写与多用户体系

**状态**：in_progress  
**决策依据**：D-013、C-007  
**目标**：以 Go 构建后端，原生纳入多用户账户隔离、公共模型池 token 计量、DeepSeek 定制化优化与 i18n 多语言，最终交付支持多平台多架构的单静态二进制 Agent Harness。

## 设计边界

- 参考 DeepSeek-Reasonix（MIT，见 C-007）的 Go 架构模式：config-driven provider 声明、MCP 兼容 plugin 体系、cache-aware context maintenance 与 `CGO_ENABLED=0` 单二进制分发。不复制其源代码、品牌、图标、文案或配置文件。
- 核心契约（任务、动作、工件、状态、预算、权限、验证、策略）以 Go struct + 验证标签实现，保持领域语义不变；`internal/harness/` 核心继续领域无关。
- 多用户隔离与 token 计量从首日纳入 Go 版本。
- DeepSeek 作为一等 provider preset，参考 Reasonix 的 prefix cache 稳定会话与 stale 输出剪枝做定制化；其他 OpenAI 兼容端点保持配置项身份。
- 强教师隔离（D-010）、密钥不落库（task13）、工件可回放（G3）与验证器驱动恢复（D-004）在 Go 版本继续生效。
- 前端使用 Vue 3 + TypeScript（D-014），不再使用 React。
- 支持 Windows、Linux、macOS 三种操作系统与 x86（amd64）、ARM（arm64）、龙芯（loong64）、RISC-V（riscv64）四种 CPU 架构；构建产物以 `CGO_ENABLED=0` 单静态二进制分发。
- Python 代码已删除（D-014），Go 版本需独立完成全部迁移与验证。

## 架构设计

### 目录结构（Go）

```
.
├── cmd/
│   ├── server/              # HTTP API 服务入口
│   └── cli/                 # 管理 CLI（账户、迁移、基线）
├── internal/
│   ├── harness/             # 领域无关核心（契约、状态、工件、验证、策略、运行时）
│   │   ├── contracts.go     # TaskContract, Action, Artifact, Budget, Permission
│   │   ├── runtime.go       # HarnessRuntime, B0-B4 策略
│   │   ├── storage.go       # SQLite 存储与可寻址工件
│   │   ├── validators.go   # 验证器注册与聚合
│   │   └── errors.go        # HarnessError, 错误码
│   ├── workbench/           # 应用层（账户、项目、会话、供应商、计量）
│   │   ├── accounts.go      # 账户、会话、鉴权中间件
│   │   ├── workspace.go     # 项目、会话、任务草案、SSE
│   │   ├── providers.go     # 供应商档案、激活、健康检查
│   │   ├── usage.go         # token 计量、公共池聚合
│   │   └── i18n.go          # 后端错误消息本地化
│   ├── providers/           # OpenAI 兼容 provider + DeepSeek 定制
│   │   ├── provider.go      # 通用 OpenAI 兼容调用
│   │   └── deepseek.go      # DeepSeek prefix cache 优化、context maintenance
│   └── skills/              # Skill 注册与运行时适配
│       ├── registry.go
│       └── adapters/        # MCP 兼容 plugin 适配
├── web/                     # 前端（Vue 3 + TypeScript + i18n）
├── configs/
│   └── harness.toml         # config-driven provider/skill/agent 声明
├── go.mod
├── go.sum
└── Makefile                 # build / cross / test
```

### 多用户隔离

- `accounts` 表：`account_id, display_name, status, locale, created_at`。
- 现有领域表（projects, conversations, provider_profiles, task_drafts, messages, runs, artifacts, events）全部新增 `account_id` 外键；所有查询按当前会话账户过滤。
- 供应商档案分 `scope: account | system`：account 档案仅属主账户可见；system 档案（built_in）所有账户可用，但用量归属调用方账户。
- 鉴权中间件：当前实现读 HttpOnly 会话 cookie 注入 `account_id`；预留可替换为 JWT/OAuth 的接口边界。
- 跨账户访问返回 404 而非 403，不泄露资源存在性。
- 迁移：首次启动建默认账户，把现有数据归到默认账户下。

### token 计量与公共池视图

- `usage_events` 表：`event_id, account_id, provider_id, model_id, run_id, conversation_id, input_tokens, output_tokens, estimated_cost_usd, latency_ms, occurred_at`。
- 每次 provider 调用完成后同步落账；失败调用只记录 error_code 不计 token。
- 公共池视图：按 `account_id × provider_id × 时间窗口` 聚合，支持今日/本周/本月与累计。
- 前端新增用量面板：显示各账户在公共模型池上的 token 消耗、估算成本与调用次数。

### DeepSeek 定制化

- DeepSeek 作为 config 预设 provider，`base_url` 默认 `https://api.deepseek.com/v1`。
- prefix cache 稳定会话：system prompt 与稳定上下文段保持固定前缀顺序，避免 cache miss。
- stale 输出剪枝：工具结果超过 TTL 或被后续工具覆盖时，在 summary compaction 前剪枝。
- context maintenance：会话注入小型稳定环境摘要，保留 cache 命中率。
- 其他 OpenAI 兼容端点保持通用调用路径，不享受 DeepSeek 定制优化。

### i18n 多语言

- 前端：引入 `vue-i18n`，抽取现有中文文案到 `zh-CN.json`，新增 `en.json`。
- 后端：错误消息支持 locale 键，按账户 `locale` 偏好返回本地化错误。
- locale 切换持久化到账户偏好；未登录时用浏览器 `Accept-Language`。
- 范围限界：只做 UI 文案与后端错误消息本地化；不做模型 system prompt 的语言切换。

## 子任务表

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task14.0 | 冻结 Go 架构边界与目录结构 | completed | D-013、C-007、本 Spec 与 `go.mod`、`internal/harness/` 目录骨架已就绪 |
| task14.1 | 核心契约与存储迁移 | completed | Go struct 覆盖全部契约；SQLite schema 对等；21 个 Go 测试通过；CGO_ENABLED=0 编译与 vet 通过 |
| task14.2 | provider 层与 DeepSeek 定制化 | pending | OpenAI 兼容 provider 调用可审计；DeepSeek preset 具备 prefix cache 稳定会话与 stale 输出剪枝；健康检查与 token 读取正确 |
| task14.3 | 多用户账户与数据隔离 | pending | accounts 表、会话中间件、account_id 过滤、scope 区分与 404 不泄露边界均有验证 |
| task14.4 | token 计量与公共池视图 | pending | usage_events 落账、按账户/供应商/时间聚合、前端用量面板与成本显示均通过测试 |
| task14.5 | workbench 应用层迁移 | pending | 项目、会话、任务草案、SSE 事件流、Skill 配置与供应商 CRUD 在 Go 版本对等可用 |
| task14.6 | 验证器与运行时基线 | pending | B0-B4 策略、验证器注册聚合、定向恢复与预算停止在 Go 版本通过对照测试 |
| task14.7 | Skill adapter 与首个 Skill 迁移 | pending | literature_search Skill 通过 Go adapter 完成搜索、归档、证据与导出；file_organizer 回归样例通过 |
| task14.8 | 前端对接与 i18n | pending | Vue 3 前端对接 Go 后端；i18n 中英双语可用；账户切换、用量面板与语言切换入口完成 |
| task14.9 | 多平台多架构与发布验证 | pending | CGO_ENABLED=0 单静态二进制在 Windows/Linux/macOS × amd64/arm64/loong64/riscv64 全组合可交叉编译；B0-B4 基线在 Go 版本通过；发布前 CI 覆盖至少 4 组合 |

## 验收

1. 单条 `make build` 产出 `CGO_ENABLED=0` 单静态二进制，跨平台编译通过。
2. `make cross-compile` 在 Windows/Linux/macOS × amd64/arm64/loong64/riscv64 全组合可交叉编译。
3. 多用户可注册、登录、切换；不同账户的项目、会话、供应商档案互不可见；跨账户访问返回 404。
4. 公共模型池视图显示各账户在系统级供应商上的 token 消耗、估算成本与调用次数；account 级供应商档案仅属主可见。
5. DeepSeek provider 具备 prefix cache 稳定会话与 stale 输出剪枝；其他 OpenAI 兼容端点通过通用路径调用。
6. 中英双语 UI 可切换并持久化到账户偏好；后端错误消息按 locale 返回。
7. B0-B4 基线与验证器在 Go 版本通过独立测试。
8. `internal/harness/` 核心继续领域无关；论文、PDF、DOI、arXiv 等领域能力仍位于 Skill 层。

## 验证证据

### task14.0 + task14.1（2026-08-01）

- `go.mod` 初始化，module `slim-agent`，Go 1.26.5。
- `internal/harness/contracts.go`：覆盖 RunStatus、RiskLevel、RecoveryLabel、Permission、Budget、Policy、TaskContract、ActionContract、ArtifactInput、Artifact、ErrorRecord、RunState、ValidatorResult、SkillManifest、Event 全部契约，验证逻辑在构造函数中执行。
- `internal/harness/errors.go`：HarnessError 与错误码常量。
- `internal/harness/storage.go`：HarnessStore 实现 Initialize、CreateRun、GetRun、ListRuns、StateVersions、TransitionRun（状态机+版本守卫）、AppendEvent、EventsAfter、PutArtifact（内容寻址+预算检查+父工件校验）、GetArtifact、ListArtifacts、ArtifactDependencyGraph、ReadArtifact（完整性校验）、RecordValidatorResult、ValidatorResults、RecordError、Errors、ReplayManifest、WriteReplayManifest。
- SQLite schema 对等（runs/state_versions/events/artifacts/validator_results/errors 表+索引）。
- 依赖：modernc.org/sqlite（纯 Go，支持 CGO_ENABLED=0）。
- 测试：21 个 Go 测试全部通过（`go test ./internal/harness/ -v -count=1`）。
- 编译：`CGO_ENABLED=0 go build` 与 `go vet` 均通过。
- 交叉编译：linux/loong64、linux/riscv64、darwin/arm64、linux/arm64、windows/arm64 全部通过。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task14，冻结 Go 重写架构边界、多用户隔离、token 计量、DeepSeek 定制与 i18n 设计 |
| v0.2 | 2026-08-01 | D-014：删除全部 Python 代码不做存档；前端切换到 Vue 3 + TypeScript；新增多平台多架构支持（Windows/Linux/macOS × amd64/arm64/loong64/riscv64）；移除 Python 对照基线；task14.9 改为多平台多架构与发布验证 |
