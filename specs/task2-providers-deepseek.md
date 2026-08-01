# task2：Provider 层与 DeepSeek 定制

**状态**：completed  
**目标**：实现 OpenAI 兼容的 Provider 层，包含通用调用、DeepSeek 定制化、调用审计、健康检查与密钥环境变量读取。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task2.0 | Provider 接口与配置契约 | completed | Provider 接口、ProviderConfig、ChatRequest/ChatResponse/Usage 结构体与校验函数就绪 |
| task2.1 | 通用 OpenAI 兼容调用 | completed | OpenAICompatibleProvider 使用 net/http + context.Context，支持超时取消、429 检测、非法 JSON 修复 |
| task2.2 | 调用审计 | completed | SQLiteAuditor 记录 token/成本/延迟/错误码到 provider_call_log 表 |
| task2.3 | 健康检查 | completed | HealthCheck 3 秒超时，返回 HealthResult{OK, LatencyMs, Error} |
| task2.4 | DeepSeek 定制化 | completed | prefix cache 稳定前缀、stale 输出剪枝、context maintenance 注入环境摘要 |
| task2.5 | 密钥环境变量读取 | completed | ResolveAPIKey 从 HARNESS_* 环境变量读取，不落库不进日志 |
| task2.6 | 预设配置 | completed | SiliconFlow、ModelScope、本地端点、DeepSeek 四个 preset |
| task2.7 | 测试覆盖 | completed | 22 个测试覆盖正常调用、超时、429、非法 JSON、prefix cache、stale 剪枝、健康检查、审计 |
| task2.8 | 鲁棒性与死代码检查 | completed | 风格一致、无死代码、并发安全（DeepSeekProvider 加 RWMutex） |

## 验收条件

1. Provider 接口统一，所有 OpenAI 兼容端点通过同一接口调用。
2. DeepSeek preset 具备 prefix cache 稳定会话与 stale 输出剪枝。
3. 密钥仅从环境变量读取，不落库、不进日志。
4. 调用审计记录 token、成本、模型、延迟、错误码。
5. 健康检查 3 秒超时，返回连通性与延迟。
6. 代码风格与 internal/harness/ 一致，无死代码。
7. 所有 HTTP 调用接受 context.Context，无裸 goroutine 泄漏。

## 验证证据

### 实现文件
- `internal/providers/provider.go`：Provider 接口、ProviderConfig、ChatMessage、ChatRequest、Usage、ChatResponse、ProviderPreset 枚举与校验。
- `internal/providers/openai.go`：OpenAICompatibleProvider，使用 net/http + context.Context，429 限流、超时取消、非法 JSON 修复。
- `internal/providers/deepseek.go`：DeepSeekProvider，嵌入 OpenAICompatibleProvider，prefix cache 稳定前缀、stale 输出剪枝、context maintenance，sync.RWMutex 并发保护。
- `internal/providers/health.go`：HealthCheck 函数，3 秒超时，返回 HealthResult。
- `internal/providers/audit.go`：AuditRecord、Auditor 接口、SQLiteAuditor，provider_call_log 表。
- `internal/providers/secrets.go`：ResolveAPIKey，从 HARNESS_* 环境变量读取。
- `internal/providers/presets.go`：PresetConfig，四个预设配置。

### 测试文件
- `internal/providers/provider_test.go`：4 个测试。
- `internal/providers/openai_test.go`：5 个测试（httptest mock）。
- `internal/providers/deepseek_test.go`：5 个测试。
- `internal/providers/health_test.go`：5 个测试。
- `internal/providers/audit_test.go`：3 个测试。

### 验证结果
- `go test ./internal/... -count=1`：harness 21 passed + providers 22 passed = 43 passed。
- `go vet ./...`：无警告。
- `go build ./...`：通过。
- `CGO_ENABLED=0` 编译支持。

### 鲁棒性检查修复
- 删除 3 处死代码：DeepSeekProvider 的 staleTTL 字段、defaultStaleTTL 常量、PruneStale 的 now 参数。
- 修复 1 处并发安全问题：DeepSeekProvider 添加 sync.RWMutex 保护 stablePrefix 和 contextSummary。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 完成 task2 全部子任务，43 个测试通过 |
