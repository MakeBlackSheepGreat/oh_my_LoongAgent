# task4：token 计量与公共池视图

**状态**：completed  
**目标**：在 workbench 应用层建立 token 计量体系，每次 provider 调用完成后按 account_id 落账，提供按账户/供应商/时间窗口的聚合查询与公共模型池视图，为前端用量面板（task8）提供数据 API。

## 输入

- task2（Provider 层）已就绪：`internal/providers/audit.go` 提供 `provider_call_log` 表（provider 层审计，领域无关，无 account_id）与 `AuditRecord`（ProviderID/ModelID/RunID/InputTokens/OutputTokens/EstimatedCostUSD/LatencyMs/ErrorCode/OccurredAt）。
- task3（多用户账户与数据隔离）需先完成：`internal/workbench/middleware.go` 的 `AccountFromContext(ctx)` 提供 account_id 注入；`internal/workbench/provider_profiles.go` 提供 scope（account|system）区分。
- 用户需求：公共大模型池显示各账号 token 消耗。
- 架构约束（D-013、D-014）：`internal/harness/` 保持领域无关；account 维度属于 workbench 应用层。

## 输出

- `internal/workbench/usage.go`：UsageEvent 契约、usage_events 表、MeterRecorder、聚合查询、公共池视图。
- `internal/workbench/usage_test.go`：覆盖落账、聚合、公共池归属、跨账户隔离、失败调用处理。
- `cmd/server/`（可能扩展）：挂载用量 API handler（受 AuthMiddleware 保护）。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task4.0 | UsageEvent 契约与 usage_events 表 | completed | UsageEvent 结构体（EventID/AccountID/ProviderID/ModelID/RunID/ConversationID/InputTokens/OutputTokens/EstimatedCostUSD/LatencyMs/ErrorCode/OccurredAt）+ Validate + InitUsageTable 幂等建表（含 account_id+occurred_at 复合索引、provider_id 索引） |
| task4.1 | MeterRecorder 计量落账 | completed | 实现 RecordUsage(ctx, rec) error；从 ctx 读取 account_id（AccountFromContext）；provider 调用成功落账 token/成本/延迟；失败调用仅记 ErrorCode 不计 token；单次 INSERT O(1) |
| task4.2 | 聚合查询 | completed | 实现 Aggregate(ctx, accountID, window) (*UsageAggregate, error)；支持 today/week/month/all 四种时间窗口；按 account × provider 聚合 input_tokens/output_tokens/total_tokens/estimated_cost/call_count；查询用索引 O(log n) |
| task4.3 | 公共池视图 | completed | 实现 PublicPoolSummary(ctx, accountID) ([]ProviderUsage, error)；汇总 system scope 供应商档案被当前账户调用的用量；account scope 档案仅属主可见；用量归属调用方账户而非档案属主 |
| task4.4 | 用量 API handler | completed | 定义 UsageHandler 结构体与 Routes 注册函数（GET /api/usage/aggregate 与 /api/usage/public-pool）；从 context 读 account_id；跨账户查询返回 404；路由挂载点预留，待 task5 建 server 后挂载 |
| task4.5 | 测试覆盖 | completed | usage_test.go 覆盖落账、四种时间窗口聚合、公共池归属、跨账户 404、失败调用不记 token、system scope 用量归属调用方；15 个测试全部通过 |
| task4.6 | 鲁棒性与死代码检查 | completed | internal/workbench/usage.go 与 audit.go/accounts.go 风格一致；无未使用导出符号/不可达分支/未引用包；所有 DB 操作用 ExecContext/QueryRowContext；无裸 goroutine；CGO_ENABLED=0 build + vet 通过 |

## 验收条件

1. usage_events 表就绪，含 account_id + occurred_at 复合索引与 provider_id 索引。
2. 每次 provider 调用完成后同步落账；失败调用仅记 ErrorCode 不计 token。
3. 聚合查询支持 today/week/month/all 四种时间窗口，按 account × provider 维度。
4. 公共池视图汇总 system scope 供应商被当前账户调用的用量；account scope 档案仅属主可见。
5. system scope 档案用量归属调用方账户（而非档案属主）。
6. 用量 API 受 AuthMiddleware 保护；跨账户查询返回 404。
7. 代码风格与 internal/harness/、internal/providers/ 一致；时间复杂度小（查询用索引）；并发效率高（共享状态用 sync.Mutex）。
8. `CGO_ENABLED=0 go build` 与 `go vet` 通过；全部测试通过。

## 设计边界

- **职责分层**：`internal/providers/audit.go` 的 `provider_call_log` 是 provider 层原始审计（领域无关，无 account_id）；`internal/workbench/usage.go` 的 `usage_events` 是 workbench 应用层计量（带 account_id 维度）。两者不合并，避免领域无关层反向依赖 account 体系。
- **落账时机**：provider 调用完成后同步落账（非异步）；不阻塞调用方主流程的错误用 best-effort 记录。
- **公共池归属**：system scope 档案所有账户可用，但 token 消耗归属调用方账户而非档案属主——这是"公共模型池显示各账号 token 消耗"的语义。
- **失败调用**：只记 ErrorCode 与 latency，不记 token；不计入聚合 token 但计入 call_count（可选，待实现时确认）。
- **时间窗口**：today/week/month/all 基于 UTC 计算；week 按自然周（周一起）或 ISO 周（待实现时确认）。
- **成本估算**：复用 task2 AuditRecord.EstimatedCostUSD；估算逻辑在 provider 层，计量层只记录不重算。

## 子任务依赖

- task4.0 依赖 task3.0（AccountFromContext）+ task3.2（workbench store）
- task4.1 依赖 task4.0 + task3.4（AuthMiddleware 注入 account_id）
- task4.2 依赖 task4.0（usage_events 表）
- task4.3 依赖 task4.0 + task3.6（provider_profiles scope）
- task4.4 依赖 task4.2 + task4.3 + task3.4（AuthMiddleware）
- task4.5 依赖 task4.1-task4.4 全部完成
- task4.6 依赖 task4.5（测试通过后做最终检查）
- task4.2 与 task4.3 可部分并行

## 影响范围

- Affected specs: task5（workbench 应用层需调用 MeterRecorder）、task8（Vue 前端用量面板消费 /api/usage/* API）。
- Affected code: `internal/workbench/usage.go`（新建）、`cmd/server/`（挂载用量路由）。
- 依赖 task3：account_id 注入与 provider_profiles scope。

## 验证证据

### 实现文件
- `internal/workbench/usage.go`：UsageEvent 契约 + Validate、InitUsageTable（幂等建表，account_id+occurred_at 复合索引 + provider_id 索引）、MeterRecorder、RecordUsage（ctx/rec 双路 account_id，自动生成 EventID，失败调用不计 token）、windowStart（today/week 自然周/month/all）、parseUsageWindow、Aggregate（GROUP BY provider_id, model_id，只统计成功调用）、PublicPoolSummary（system scope 供应商用量归属调用方）。
- `internal/workbench/handlers_usage.go`：UsageHandler + Routes 注册（GET /api/usage/aggregate、GET /api/usage/public-pool）、writeJSON、writeError、httpStatusFor（HarnessError.Code → HTTP 状态码映射）。

### 测试文件
- `internal/workbench/usage_test.go`：15 个测试：
  - 契约校验：TestUsageEvent_Validate（合法 + 空 account_id + 负 token）。
  - 落账：TestRecordUsage_Success（自动 EventID + 字段持久化）、TestRecordUsage_AccountFromContext（ctx 注入）、TestRecordUsage_NoAccount（无 account 错误）、TestRecordUsage_FailureCall（失败不计入聚合）。
  - 聚合：TestAggregate_All（按 provider 拆分 + 总量）、TestAggregate_Today（窗口过滤）、TestWindowStart（today/week 周一/month 边界）。
  - 公共池：TestPublicPoolSummary（system scope 归属调用方 + 私有 provider 不入池）、TestPublicPoolSummary_NoSystemProfiles（空返回）。
  - Handler：TestUsageHandler_Aggregate（端到端）、TestUsageHandler_Aggregate_Unauthorized（401）、TestUsageHandler_Aggregate_CrossAccount（404）、TestUsageHandler_Aggregate_InvalidWindow（400）、TestUsageHandler_PublicPool（端到端）。
  - 映射：TestHttpStatusFor（全部错误码无遗漏）。

### 验证结果
- `go test ./internal/workbench/ -v -count=1`：47 个测试全部通过（32 个 task3 + 15 个 task4）。
- `go test ./internal/... -count=1`：harness 21 + providers 22 + workbench 47 = 90 个测试全部通过。
- `CGO_ENABLED=0 go build ./...`：通过。
- `go vet ./...`：无警告。

### 设计确认
- 时间窗口：week 按自然周（周一起）UTC。
- 失败调用：只记 ErrorCode 与 LatencyMs，不计 token；聚合（error_code = ''）不计入 call_count。
- task4.4 仅定义 handler 与路由注册函数，挂载点预留在 `cmd/server`（task5 建立后调用 `UsageHandler.Routes(mux)` 挂载）。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task4 spec；与 task2 provider_call_log 职责分层；拆分 7 个子任务 task4.0-task4.6 |
| v1.0 | 2026-08-01 | task4 全部子任务完成；15 个新测试通过（全项目 90 个）；CGO_ENABLED=0 build + vet 通过 |
