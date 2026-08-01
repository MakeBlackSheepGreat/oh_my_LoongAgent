# task5：workbench 应用层 HTTP 服务

**状态**：completed  
**目标**：在 Go 后端建立完整的 workbench 应用层 HTTP 服务，把 task3 已就绪的账户、会话、鉴权、隔离与供应商档案能力以 RESTful API 暴露给前端（task8）与本地用户；补齐 conversations / messages / task_drafts 的领域 CRUD；实现任务草案审批工作流（D-011：approve 后调用 HarnessStore.CreateRun）；提供 SSE 事件流；用 embed.FS 把 Vue 前端构建产物嵌入单静态二进制，使 `./slim-agent serve` 单命令即可启动完整工作台。

## 输入

- task1（Go 核心契约与存储）已就绪：`internal/harness/` 提供 HarnessStore、TaskContract、RunState、Event、AppendEvent、CreateRun、EventsAfter。
- task2（Provider 层与 DeepSeek 定制）已就绪：`internal/providers/` 提供 ProviderConfig、HealthCheck（3 秒超时）、ResolveAPIKey（环境变量密钥）。
- task3（多用户账户与数据隔离）已就绪：`internal/workbench/` 提供 WorkbenchStore、Account、Session、AuthMiddleware（HttpOnly cookie + AccountFromContext）、AccountScoped（Project CRUD + 跨账户 nil,nil）、ProviderProfile CRUD + scope 区分 + ActivateProfile。
- 用户需求：项目、会话、任务草案、SSE 事件流、供应商档案 CRUD 在 Go 版本可用（SPEC.md 主任务表 task5 退出条件）。
- 架构约束（D-011、D-013、D-014、D-007）：模型消息先形成可见答复和结构化草案，用户批准后才能创建运行；`internal/harness/` 保持领域无关；Go 唯一后端；多平台多架构 + CGO_ENABLED=0 单静态二进制。
- SOUL.md 代码口味：契约先行、状态显式、模块小而聚焦、核心入口短小、失败路径属于正式产品能力。

## 输出

- `cmd/server/main.go`：HTTP 服务入口，flag/env 配置装配、优雅关闭、静态文件嵌入。
- `internal/workbench/server.go`：Server 结构体、路由注册、统一 JSON 错误响应、CORS、HarnessError → HTTP 状态码映射。
- `internal/workbench/conversations.go`：Conversation 契约 + CRUD（Create/Get/List/UpdateTitle/Delete）。
- `internal/workbench/messages.go`：Message 契约 + AppendMessage/ListMessages。
- `internal/workbench/task_drafts.go`：TaskDraft 契约 + CRUD + ApproveDraft（调用 HarnessStore.CreateRun）/ RejectDraft。
- `internal/workbench/isolation.go`：补 UpdateProject/DeleteProject。
- `internal/workbench/events.go`：EventHub（订阅/广播，sync.RWMutex 保护 subscribers）。
- `internal/workbench/handlers_*.go`：认证、账户、项目、会话、消息、草案、供应商、SSE、健康检查 handler。
- `internal/workbench/*_test.go` 与 `cmd/server/main_test.go`：端到端 HTTP 测试（httptest.NewServer + 真实 store）。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task5.0 | HTTP 服务入口与配置装配 | completed | `cmd/server/main.go` 解析 flag（--addr、--data-dir、--cors-origins、--auth-bypass）与 env（HARNESS_DATA_DIR、HARNESS_API_PORT）；装配 HarnessStore + WorkbenchStore + AuthMiddleware；调用 InitAll + EnsureDefaultAccount；`/health` 返回 200；SIGINT/SIGTERM 触发 5 秒超时优雅关闭 |
| task5.1 | 统一响应层与错误映射 | completed | `internal/workbench/server.go` 的 Server 结构体；公开/受保护双 mux 路由注册；writeJSON/writeError（复用 task4）；HarnessError.Code → HTTP 状态码映射表（httpStatusFor）；CORS 中间件按白名单严格匹配 |
| task5.2 | 认证 API | completed | `POST /api/auth/login` 选择账户 → CreateSession → Set-Cookie HttpOnly SameSite=Strict Path=/；`POST /api/auth/logout` DeleteSession + Clear cookie；`GET /api/auth/me` 从 context 返回 Account；WithAuthBypass dev 模式 |
| task5.3 | 领域模型扩展 | completed | conversations.go（Create/Get/List/UpdateTitle/Delete 事务级联删消息）；messages.go（AppendMessage/ListMessages 分页）；task_drafts.go（Create/Get/List/UpdateObjective/Delete + DraftService approve/reject）；isolation.go 补 UpdateProject/DeleteProject；全部 AccountScoped 方法跨账户 nil,nil |
| task5.4 | 项目与会话 API | completed | `GET/POST /api/projects`；`GET/PATCH/DELETE /api/projects/{id}`；`GET/POST /api/conversations`；`GET/PATCH/DELETE /api/conversations/{id}`；`GET/POST /api/conversations/{id}/messages`；所有 handler 用 AccountFromContext；跨账户 404；body 校验失败 400 |
| task5.5 | 任务草案 API 与审批工作流 | completed | `GET/POST /api/task-drafts`；`GET/PATCH/DELETE /api/task-drafts/{id}`；`POST /api/task-drafts/{id}/approve`：draft→approved，调用 harness.NewTaskContract + HarnessStore.CreateRun，返回 run_id；`POST /api/task-drafts/{id}/reject`；非 draft 状态 approve 返回 409 |
| task5.6 | 供应商档案与健康检查 API | completed | `GET/POST /api/providers`（仅 account scope）；`DELETE /api/providers/{id}`（system scope 不可删）；`POST /api/providers/{id}/activate` 复用 ActivateProfile；`GET /api/providers/{id}/health` 复用 providers.HealthCheck（3 秒超时，不记 token，不调 Chat） |
| task5.7 | SSE 事件流 | completed | events.go 的 EventHub（buffered chan 容量 16 + sync.RWMutex；慢消费者丢事件写 event_dropped）；`GET /api/events` 全局流（AuthMiddleware 保护，按账户过滤：task_drafts.run_id 反查属主）；客户端断开（ctx.Done()）自动清理；帧格式 `event: <kind>\ndata: <json>\n\n` |
| task5.8 | 静态文件嵌入与 SPA fallback | completed | 根级 webdist.go `//go:embed all:web/dist`（跨目录 embed 限制的解）；spaHandler 路由 /api/* 到后端、其余静态文件、缺失 fallback index.html；Content-Type 由 http.FileServer 处理 |
| task5.9 | 测试覆盖 | completed | `cmd/server/main_test.go` 11 个端到端测试（httptest.NewServer + 内存 SQLite + 文件 harness + 完整 spaHandler 组合）：health、auth flow、项目 CRUD、会话+消息、跨账户 404、草案 approve→CreateRun、草案 reject、供应商 CRUD+激活+health、SSE 订阅按账户过滤、SPA fallback、两账户并发创建互不干扰 |
| task5.10 | 鲁棒性与死代码检查 | completed | 修复：skill_id 为空 approve 校验失败（CreateDraft 默认 generic）；:memory: 多连接分裂（SetMaxOpenConns(1)）；goroutine 中 t.Fatal（错误通道收集）；删除 net/url/min 冗余。CGO_ENABLED=0 build + vet 通过；4 组合交叉编译通过 |

## 验收条件

1. `cmd/server/main.go` 可启动 HTTP 服务，监听 `127.0.0.1:8000`；`CGO_ENABLED=0 go build ./cmd/server/` 通过。
2. 全部 API 受 AuthMiddleware 保护；未登录返回 401 JSON；跨账户访问返回 404 不泄露存在性。
3. 认证流程完整：login → Set-Cookie HttpOnly SameSite=Strict Path=/ → `/api/auth/me` 返回 Account → logout 清除 cookie。
4. 项目、会话、消息、任务草案、供应商档案均有完整 CRUD（GET 列表/POST 创建/GET 单个/PATCH 更新/DELETE）。
5. 任务草案 approve 后调用 `harness.NewTaskContract` + `HarnessStore.CreateRun` 创建 RunState，响应返回 `run_id`；非 draft 状态再 approve 返回 409。
6. 供应商档案 health 端点复用 `providers.HealthCheck`，3 秒超时，不记 token、不调 Chat。
7. SSE 事件流：客户端订阅后实时收到 `event:` 帧；客户端断开自动清理 subscriber；SSE goroutine 不泄漏。
8. 静态文件嵌入：单二进制启动后浏览器直接访问 `http://127.0.0.1:8000` 看到 Vue 工作台；`fs.ValidPath` 拒绝 `..`。
9. `make cross-compile` 至少通过 4 组合（windows/amd64、linux/amd64、linux/arm64、darwin/arm64）；`make all` 产出含前端的单二进制。
10. `go vet ./...` 无警告；全部测试通过（harness + providers + workbench + cmd/server）。
11. 代码风格与 internal/harness/、internal/providers/、internal/workbench/ 一致；时间复杂度小（查询用索引，O(log n) 或 O(1)）；并发效率高（共享状态用 sync.RWMutex，SSE 广播用 RLock）。

## 设计边界

- **职责分层**：`internal/harness/` 保持领域无关；HTTP 服务入口、路由、handler、SSE Hub 属于 `internal/workbench/` 应用层；`cmd/server/main.go` 仅做装配与启动，不含业务逻辑。
- **不实现范围**：task4 计量（usage_events 落账）、BVAR 路由、Skill 运行时、Vue 具体页面。task5 在 handler 层预留 `MeterRecorder` 接入点（如 Provider Chat 调用前后注入），task4 完成后挂上即可。
- **审批工作流**：D-011 "模型消息必须先形成可见答复和结构化草案，用户批准后才能创建运行"。task_drafts.status 取值 `draft|approved|rejected`；approve 时调用 `harness.NewTaskContract` + `HarnessStore.CreateRun`；拒绝不创建运行。
- **Skill 配置**：主任务表写 "Skill 配置"，但 `internal/skills/` 当前未建。task5 范围内只提供 `/api/skills` 占位 handler（返回空列表），具体 Skill 注册留给 task7。
- **SSE 实现策略**：workbench 层 EventHub 订阅 harness.AppendEvent 的返回值（已返回 `*Event`）；hub 用 buffered chan（容量 16）+ sync.RWMutex 保护 subscribers map；慢消费者丢弃事件并写一条 `event_dropped` 警告。
- **静态文件嵌入**：用 `//go:embed all:web/dist`（注意 `all:` 前缀以包含 `_` 开头文件如 `assets/`）；web/dist 缺失时 `go build` 失败，Makefile 的 `make all` 先跑 `make frontend` 再 `make build`；开发模式仍用 `npm run dev` + Vite 代理。
- **CORS**：开发期靠 Vite 代理（vite.config.ts 已配置 `/api`、`/health` 到 127.0.0.1:8000）；生产期靠 embed.FS 同源，无需 CORS；`HARNESS_CORS_ORIGINS` 仅在浏览器绕过 Vite 代理（如独立前端域名）时启用，按白名单严格匹配。
- **错误响应统一格式**：`{"error": "<CODE>", "detail": "<message>"}`；与 task3 middleware.go 的 `writeUnauthorized` 一致；HarnessError.Code → HTTP 状态码映射集中在 `internal/workbench/server.go` 的 `httpStatusFor(code)` 函数。
- **跨账户边界**：所有 handler 用 `AccountFromContext(r.Context())` 取 account_id；AccountScoped 方法跨账户返回 nil,nil → handler 映射为 404；system scope 供应商档案跨账户可见但不可删（403）。
- **优雅关闭**：捕获 SIGINT/SIGTERM；`http.Server.Shutdown(ctx)` 用 5 秒超时等待在途请求；SSE 连接靠 ctx.Done() 主动断开。
- **配置优先级**：flag > env > 默认值。默认 `--addr 127.0.0.1:8000`、`--data-dir .harness-data`；env `HARNESS_API_PORT` 覆盖端口。
- **密钥处理**：Handler 不直接读环境变量密钥；供应商 health 调用 `providers.HealthCheck`，内部用 `ResolveAPIKey`；Chat 调用通过 Provider 实例（task5 不实现 Chat handler，留给 task6+）。
- **路径安全**：静态文件服务用 `fs.FS` + `http.FileServerFS`（Go 1.22+）或 `http.FileServer(http.FS(embed.FS))`；`fs.ValidPath` 自动拒绝 `..`；不暴露文件系统绝对路径。

## 子任务依赖

- task5.0 是入口，最先完成（其他 handler 需要挂载点）。
- task5.1 依赖 task5.0（路由注册）。
- task5.2 是纯领域层扩展，可与 task5.1 并行。
- task5.3 依赖 task5.0 + task5.1 + task5.2（领域层 + 响应层）。
- task5.4 依赖 task5.0 + task5.1 + task5.2（领域层）。
- task5.5 依赖 task5.0 + task5.1 + task5.2 + task1（HarnessStore.CreateRun）。
- task5.6 依赖 task5.0 + task5.1 + task2（providers.HealthCheck）+ task3.6（ProviderProfile CRUD）。
- task5.7 依赖 task5.0 + task5.1 + task1（events）+ task5.2（account 过滤）。
- task5.8 依赖 task5.0 + web/dist 构建产物。
- task5.9 依赖 task5.1-task5.8 全部完成。
- task5.10 依赖 task5.9（测试通过后做最终检查）。
- task5.2 与 task5.1 可部分并行；task5.4/5.5/5.6/5.7 在 task5.2 完成后可部分并行。

## 影响范围

- Affected specs: task4（MeterRecorder 接入点预留）、task6（验证器运行时复用 SSE Hub）、task7（Skill adapter 挂载 /api/skills）、task8（Vue 前端消费全部 /api/* + SSE + 静态文件嵌入）。
- Affected code: `cmd/server/`（新建 main.go）、`internal/workbench/`（新增 server.go、conversations.go、messages.go、task_drafts.go、events.go、handlers_*.go）、`Makefile`（`make all` 依赖 `make frontend`）、`web/dist`（embed.FS 目标，需 `npm run build` 产出）。
- 依赖 task1（HarnessStore）、task2（providers）、task3（workbench 应用层）。
- 不影响 `internal/harness/` 与 `internal/providers/` 现有代码（只读引用）。

## 验证证据

### 实现文件
- `cmd/server/main.go`：flag/env 装配（--addr/--data-dir/--cors-origins/--auth-bypass；HARNESS_DATA_DIR/HARNESS_API_PORT）、WorkbenchStore + HarnessStore 双文件存储、默认账户迁移、优雅关闭（SIGINT/SIGTERM + 5 秒 Shutdown）、spaHandler（/api/* 后端、静态文件、fallback index.html）。
- `webdist.go`（项目根）：`//go:embed all:web/dist` 嵌入前端产物（embed 不能跨目录，故置于根级包）。
- `internal/workbench/server.go`：Server 结构体、公开/受保护双 mux、CORS 白名单、/api/skills 占位（task7 挂接）。
- `internal/workbench/conversations.go`、`messages.go`、`task_drafts.go`：领域 CRUD + DraftService（ApproveDraft 先 CreateRun 再标记 approved；RejectDraft）。
- `internal/workbench/events.go`：EventHub（buffered chan 16 + sync.RWMutex，慢消费者丢事件写 event_dropped）。
- `internal/workbench/handlers_auth.go`、`handlers_projects.go`、`handlers_conversations.go`、`handlers_task_drafts.go`、`handlers_providers.go`、`handlers_sse.go`：全部 REST + SSE handler。
- `internal/workbench/isolation.go`：补 UpdateProject/DeleteProject。
- `internal/workbench/store.go`：task_drafts 表补 run_id 列。

### 测试文件
- `cmd/server/main_test.go`：11 个端到端测试（httptest.NewServer + 完整 spaHandler 组合）：
  - TestHealthEndpoint（免鉴权）、TestAuthFlow（登录→me→logout + 401 边界）。
  - TestProjectCRUD（创建/列表/查询/更新/删除全流程）、TestConversationAndMessages。
  - TestCrossAccount404（跨账户 404 + 列表隔离）、TestDraftApproveCreateRun（approve→run_id + 二次 approve 409）、TestDraftReject。
  - TestProviderProfileCRUD（创建/列表/激活/health/删除）。
  - TestSSEEvents（订阅收到按账户过滤的事件）、TestSpaFallback（index.html + 未知路由 fallback）。
  - TestConcurrentProjects（两账户 10 goroutine 并发创建互不干扰，错误通道收集）。

### 验证结果
- `go test ./... -count=1`：harness 21 + providers 22 + workbench 47 + cmd/server 11 = 101 个测试全部通过。
- `CGO_ENABLED=0 go build ./...`：通过。
- `go vet ./...`：无警告。
- 交叉编译：windows/amd64、linux/amd64、linux/arm64、darwin/arm64 全部通过。
- `make all`（frontend + build）产出含前端的单二进制。

### 鲁棒性检查修复
- 修复 skill_id 为空时 approve 校验失败：CreateDraft 默认 skill_id=generic。
- 修复 :memory: SQLite 多连接分裂：测试 db SetMaxOpenConns(1)。
- 修复 goroutine 中调用 t.Fatal：并发测试用错误通道收集。
- 删除冗余：net/url 导入、自定义 min、errValidation/errNotFound 伪函数。
- SSE 按账户过滤：task_drafts.run_id 反查属主（ApproveDraft 建立归属）。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task5 spec；拆分 11 个子任务 task5.0-task5.10；明确 D-011 审批工作流、embed.FS 静态嵌入、SSE Hub、HarnessError → HTTP 映射等设计边界；不实现 task4 计量与 task6+ 运行时，但预留接入点 |
| v1.0 | 2026-08-01 | task5 全部子任务完成；11 个端到端测试通过；CGO_ENABLED=0 build + vet + 4 组合交叉编译通过；task4 用量 API 挂载 |
