# task3：多用户账户与数据隔离

**状态**：completed  
**目标**：在 Go 后端引入账户体系与会话鉴权，所有领域表按 account_id 隔离，供应商档案区分 account/system scope，跨账户访问返回 404 不泄露存在性，为 task4 计量与 task5 workbench 应用层提供 account_id 维度地基。

## 输入

- task1（Go 核心契约与存储）已就绪：`internal/harness/` 提供 HarnessStore 与 SQLite schema。
- task2（Provider 层与 DeepSeek 定制）已就绪：`internal/providers/` 提供可审计调用与密钥环境变量读取。
- 用户需求：多账号相互隔离；公共大模型池显示各账号 token 消耗。
- 架构约束（D-013、D-014、C-007）：Go 唯一后端；`internal/harness/` 保持领域无关；账户隔离属于 workbench 应用层。
- SOUL.md 代码口味：契约先行、状态显式、模块小而聚焦、模型/工具/存储/供应商可替换。

## 输出

- `internal/workbench/accounts.go`：Account 契约、AccountStatus 枚举、Validate、ULID 生成。
- `internal/workbench/store.go`：WorkbenchStore（复用 `*sql.DB`），accounts/sessions/projects/conversations/provider_profiles/task_drafts/messages 表 schema，幂等初始化。
- `internal/workbench/session.go`：Session 结构体、crypto/rand session_id、7 天过期、CRUD。
- `internal/workbench/middleware.go`：AuthMiddleware（HttpOnly cookie）、AccountFromContext、401 边界、WithAuthBypass 开发模式。
- `internal/workbench/isolation.go`：AccountScoped 包装器、查询过滤、跨账户 nil, nil（对应 404）。
- `internal/workbench/provider_profiles.go`：ProviderProfile 契约、scope（account|system）、CRUD、ActivateProfile 单账户单 provider 单激活。
- `internal/workbench/*_test.go`：覆盖账户、会话、隔离、供应商档案、中间件。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task3.0 | Account 契约与 accounts 表 | completed | Account 结构体（AccountID/DisplayName/Status/Locale/CreatedAt）+ AccountStatus 枚举（active/disabled）+ Validate + InitAccountsTable 就绪 |
| task3.1 | Account CRUD 与默认账户迁移 | completed | CreateAccount 生成 ULID；GetAccount 不存在返回 nil,nil；ListAccounts 按 created_at 排序；EnsureDefaultAccount 首次启动建默认账户（display_name=default, locale=zh-CN） |
| task3.2 | 领域表与 account_id 外键 | completed | projects/conversations/provider_profiles/task_drafts/messages 表全部含 account_id 外键与索引；provider_profiles 含 scope 字段；InitWorkbenchSchema 幂等（IF NOT EXISTS） |
| task3.3 | 会话管理 | completed | Session 结构体 + sessions 表；CreateSession 用 crypto/rand 生成 session_id，7 天过期；GetSession 过期或不存在返回 nil；DeleteSession 删除 |
| task3.4 | AuthMiddleware | completed | Wrap(http.Handler) http.Handler；从 cookie 解析 session_id 注入 account_id 到 context；未登录返回 401 JSON；AccountFromContext 工具函数；预留 WithAuthBypass |
| task3.5 | 数据隔离查询 | completed | AccountScoped 包装 WorkbenchStore + account_id；ListProjects 等自动附加 WHERE account_id = ?；GetProject 跨账户返回 nil,nil；ListProfiles 返回 account scope + system scope |
| task3.6 | 供应商档案 CRUD | completed | ProviderProfile 结构体（ProfileID/AccountID/ProviderID/DisplayName/BaseURL/ModelID/APIKeyEnv/Scope/IsActive/CreatedAt）；Create/Get/List/Activate/Delete；ActivateProfile 同账户同 provider 单激活；密钥仍从环境变量读取不落库 |
| task3.7 | 测试覆盖 | completed | accounts_test/session_test/isolation_test/provider_profiles_test/middleware_test 全部通过；32 个测试覆盖同账户可见、跨账户 404、scope 查询、激活切换、密钥不落库、401 边界 |
| task3.8 | 鲁棒性与死代码检查 | completed | 删除未使用的 ErrNoSession 死代码；所有 DB 操作用 ExecContext/QueryRowContext；无裸 goroutine；CGO_ENABLED=0 build + vet 通过 |

## 验收条件

1. accounts 表与 Account 契约就绪，支持创建、查询、列表、默认账户迁移。
2. 会话通过 HttpOnly cookie 注入 account_id；未登录返回 401；预留 JWT/OAuth 接口边界。
3. 所有领域表（projects/conversations/provider_profiles/task_drafts/messages）含 account_id 外键；列表查询自动附加 WHERE account_id = ?。
4. 跨账户访问返回 nil, nil（对应 HTTP 404，不泄露资源存在性）。
5. 供应商档案支持 account 与 system 两种 scope；account scope 仅属主可见，system scope 所有账户可用但用量归属调用方。
6. ActivateProfile 确保同账户同 provider 只有一个激活档案。
7. 密钥从环境变量读取，不落库、不进日志（与 task2 一致）。
8. 代码风格与 internal/harness/、internal/providers/ 一致；时间复杂度小（查询用索引，O(log n) 或 O(1)）；并发效率高（共享状态用 sync.Mutex）。
9. `CGO_ENABLED=0 go build` 与 `go vet` 通过；全部测试通过。

## 设计边界

- `internal/harness/` 核心保持领域无关；账户隔离属于 `internal/workbench/` 应用层，不反向污染核心。
- 不实现完整 RBAC；当前仅区分 account scope 与 system scope。
- system scope 档案定义：built_in 供应商，所有账户可见可用，但用量归属调用方账户（task4 计量落地）。
- 鉴权中间件当前实现读 HttpOnly cookie；预留可替换为 JWT/OAuth 的接口边界。
- 跨账户访问统一返回 404 而非 403，不泄露资源存在性。
- 迁移策略：首次启动建默认账户，把现有数据归到默认账户下。

## 子任务依赖

- task3.1 依赖 task3.0（Account 契约）
- task3.2 依赖 task3.0（WorkbenchStore）
- task3.3 依赖 task3.1（CreateSession 需 accountID）
- task3.4 依赖 task3.3（AuthMiddleware 需 GetSession）
- task3.5 依赖 task3.2 + task3.4（领域表 + account_id 注入）
- task3.6 依赖 task3.2 + task3.5（provider_profiles 表 + scope 查询）
- task3.7 依赖 task3.1-task3.6 全部完成
- task3.8 依赖 task3.7（测试通过后做最终检查）
- task3.0 与 task3.2 中 SubTask 可部分并行

## 影响范围

- Affected specs: task4（计量依赖 account_id 维度）、task5（workbench CRUD 需注入 account_id）、task8（Vue 前端需账户切换入口）。
- Affected code: `internal/workbench/`（新建）、`internal/harness/storage.go`（可能扩展 account_id 字段）、`cmd/server/`（挂载 AuthMiddleware）。

## 验证证据

### 实现文件
- `internal/workbench/accounts.go`：Account 契约、AccountStatus 枚举、Validate、CreateAccount（ULID）、GetAccount（nil,nil）、ListAccounts、EnsureDefaultAccount。
- `internal/workbench/ulid.go`：自实现 ULID（crypto/rand + Crockford Base32），无外部依赖。
- `internal/workbench/store.go`：WorkbenchStore、InitAccountsTable、InitSessionsTable、InitWorkbenchSchema（projects/conversations/provider_profiles/task_drafts/messages 含 account_id 外键与索引）、InitAll。
- `internal/workbench/session.go`：Session 契约、CreateSession（crypto/rand 32 字节 hex，7 天过期）、GetSession（过期返回 nil）、DeleteSession。
- `internal/workbench/middleware.go`：AuthMiddleware、AccountFromContext、Wrap（cookie 解析 + context 注入）、401 JSON 边界、WithAuthBypass 开发模式。
- `internal/workbench/isolation.go`：AccountScoped 包装器、CreateProject、GetProject（跨账户 nil,nil）、ListProjects（WHERE account_id = ?）。
- `internal/workbench/provider_profiles.go`：ProviderProfile 契约、ProfileScope 枚举（account/system）、CreateProfile、GetProfile、ListProfiles（account + system scope）、ActivateProfile（事务内单激活）、DeleteProfile（system scope 不可删）。

### 测试文件
- `internal/workbench/store_test.go`：newTestStore helper（内存 SQLite + InitAll）。
- `internal/workbench/accounts_test.go`：9 个测试（Validate 5 子用例 + ULID 唯一性 + CRUD + 默认账户）。
- `internal/workbench/session_test.go`：6 个测试（创建 + 空 account_id + 不存在 + 查询 + 过期 + 删除）。
- `internal/workbench/isolation_test.go`：4 个测试（同账户创建/查询 + 跨账户 nil + 列表隔离）。
- `internal/workbench/provider_profiles_test.go`：8 个测试（account scope + account+system 列表 + 跨账户 nil + system 可见 + 单激活 + account 删除 + system 禁删 + Validate）。
- `internal/workbench/middleware_test.go`：5 个测试（无 cookie 401 + 无效 session 401 + 授权注入 + bypass 模式 + context 无 account 错误）。

### 验证结果
- `go test ./internal/workbench/ -v -count=1`：32 个测试全部通过（0.752s）。
- `go test ./internal/... -count=1`：harness 21 + providers 22 + workbench 32 = 75 个测试全部通过。
- `CGO_ENABLED=0 go build ./...`：通过。
- `go vet ./...`：无警告。
- 无外部依赖新增（ULID 自实现，避免 oklog/ulid 网络依赖问题）。

### 鲁棒性检查
- 删除 1 处死代码：ErrNoSession（定义但未使用）。
- 所有 DB 操作用 ExecContext/QueryRowContext/QueryContext/BeginTx，无裸 SQL。
- 无裸 goroutine 泄漏（代码中未启动 goroutine）。
- ActivateProfile 用事务保证同账户同 provider 单激活原子性。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task3 spec，基于 `.trae/specs/task3-accounts-isolation/` 草稿整合；拆分 9 个子任务 task3.0-task3.8 |
| v1.0 | 2026-08-01 | task3 全部子任务完成；32 个测试通过；自实现 ULID 替代外部依赖；删除 ErrNoSession 死代码 |
