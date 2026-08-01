# task6：验证器与运行时基线

**状态**：completed  
**目标**：在 Go 版本建立验证器体系与受限运行时：验证器注册聚合、内建验证器、受限工具执行器（权限/风险/路径/审批门）、B0-B4 计算分配策略（B4 为 BVAR 预算感知验证器路由）、预算停止与定向恢复。让确定性验证结果成为路由、修复和停止决策的依据。

## 输入

- task1（Go 核心契约与存储）已就绪：`internal/harness/contracts.go` 提供 ValidatorResult、RunState、ErrorRecord、RecoveryLabel（retry/repair/human_review/stop）、Budget、Permission、Policy、ActionContract、TaskContract；`internal/harness/storage.go` 提供 RecordValidatorResult、ValidatorResults、TransitionRun、AppendEvent、PutArtifact。
- task2（Provider 层）已就绪：`internal/providers/` 提供可审计模型调用（task6 的 B0-B4 需要 Chat 调用与 token 计量接入点）。
- task5（workbench HTTP 服务）依赖 task6 的验证器与运行时：task_drafts approve→CreateRun 后由运行时执行（task5.5 预留 CreateRun 调用，执行循环在 task6 落地）。
- 研究框架（PROJECT_GOALS.md RQ2/RQ3）：B0-B4 是计算分配策略的公平基线，BVAR 是 B4 条件下的策略而非框架边界。
- 旧主线领域语义（D-014 已删 Python，从归档追溯）：task6 工具治理、task7 验证器、task8 运行时基线、task9 B4 与经验。
- 架构约束（D-003/D-004/D-005/D-009/D-013/D-014）：`internal/harness/` 领域无关；验证器、策略、恢复属于核心；论文/PDF 等领域能力留在 Skill 层。
- SOUL.md 代码口味：契约先行、状态显式、阶段门驱动、失败状态与恢复路径属于正式产品能力。

## 输出

- `internal/harness/validators.go`：Validator 接口、Registry（注册/查找/聚合）、内建验证器。
- `internal/harness/tools.go`：受限工具执行器（Tool 接口、ToolResult、Policy 检查：权限白名单/风险等级/路径隔离/预条件/审批门/回滚标准化）。
- `internal/harness/runtime.go`：HarnessRuntime 执行循环（验证动作→执行工具→记录事件→状态转换→工件落库→计量接入点）。
- `internal/harness/strategies.go`：B0-B4 策略接口与实现（固定单流程/单 Agent 闭环/串行角色复用/固定分支候选/BVAR）。
- `internal/harness/budget.go`：预算检查与停止（MaxModelCalls/MaxToolCalls/MaxRuntimeSeconds/MaxCostUSD；超时、取消、耗尽、早停审计）。
- `internal/harness/recovery.go`：定向恢复（RecoveryLabel 分类驱动：retry/repair/human_review/stop）。
- `internal/harness/*_test.go`：验证器、工具治理、策略、预算、恢复测试。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task6.0 | 验证器接口与注册表 | completed | `validators.go` 定义 `Validator` 接口（ID/Version/Validate(ctx, run, artifacts) → *ValidatorResult）；`Registry` 支持 Register/Lookup/List/聚合（多验证器结果按 run 汇总，失败优先）；重复注册返回 CONFLICT；未知验证器返回 NOT_FOUND |
| task6.1 | 内建验证器 | completed | 工件存在性验证器（artifact_ids 均存在且哈希匹配）；Schema 验证器（工件 JSON 对 JSON Schema 校验）；引用完整性验证器（引用只能指向已知工件/证据）；预算约束验证器（usage 未超 Budget）；各验证器返回结构化 Findings 与 Confidence |
| task6.2 | 受限工具执行器 | completed | `tools.go` 定义 `Tool` 接口与 `ToolResult`；执行前 Policy 检查：工具名在白名单（AllowedToolNames）、权限在 AllowedPermissions、RiskHigh/Critical 需审批（ApprovalRequiredFor）、WorkspaceRoot 路径隔离（拒绝 `..` 逃逸）、AllowNetworkDomains 网络域白名单；越权返回 PERMISSION_DENIED；可逆动作附 RollbackSummary；错误用 HarnessError 标准化 |
| task6.3 | HarnessRuntime 执行循环 | completed | `runtime.go` 的 `HarnessRuntime`：接受已审批 RunState；循环执行"策略选动作→验证器验证→工具执行→事件记录→状态转换→工件落库"；每次模型调用/工具调用走预算检查；调用 providers 后接入 MeterRecorder（task4 落账）；TransitionRun 受版本守卫保护；执行结束（完成/失败/取消/预算耗尽）写终态事件 |
| task6.4 | B0-B3 基线策略 | completed | B0 固定单流程：一次执行无自由路由无修复（baseline）；B1 单 Agent 闭环：验证失败可一次定向修复（AGI 缺口修复：修复限制在可验证槽位）；B2 串行角色复用：路由/规划/执行/审查角色按固定顺序串行复用同一模型（D-002）；B3 固定分支候选：候选数与选择规则固定（固定多候选或固定多 Agent，无动态触发）；各策略产出可对比的调用次数/成本/验证轨迹 |
| task6.5 | B4 BVAR 路由 | completed | BVAR 依据风险等级、验证置信度、候选分歧与剩余预算选择下一动作（D-004：DRA→AAR→AGI→verifier→BVAR）；动态触发条件显式化（新增检索/候选/修复/审查/云端升级）；预算耗尽降级到确定性动作或人工审查；与 B0-B3 共享运行时、预算与审计接口（PROJECT_GOALS 架构边界） |
| task6.6 | 预算与停止 | completed | `budget.go` 实现 Budget 检查：MaxModelCalls/MaxToolCalls/MaxRuntimeSeconds/MaxCostUSD 任一超限即停止；支持 context 取消；早停（验证器全通过即提前结束）；停止原因写入事件（budget_exhausted/cancelled/early_stop）；审计记录完整 |
| task6.7 | 定向恢复 | completed | `recovery.go` 依据 ErrorRecord.Code 与 RecoveryLabel 分类导向恢复：retry（幂等重试，限次）、repair（AGI 定向修复）、human_review（RiskHigh/未知失败转人工）、stop（不可恢复/预算耗尽）；失败定位率与恢复成功率可度量 |
| task6.8 | 测试覆盖 | completed | validators_test（注册/聚合/内建验证器）、tools_test（权限拒绝/路径逃逸/审批门/网络域）、runtime_test（执行循环/状态转换/计量接入）、strategies_test（B0-B3 对照轨迹）、bvar_test（动态触发/预算降级）、budget_test（超限停止/早停/取消）、recovery_test（四类恢复路径）；全部离线可跑 |
| task6.9 | 鲁棒性与死代码检查 | completed | 风格与 harness/providers/workbench 一致；无未使用导出符号/不可达分支/未引用包；无裸 goroutine 泄漏；context 贯穿所有阻塞调用；CGO_ENABLED=0 build + vet 通过 |

## 验收条件

1. 验证器注册表支持注册、查找、聚合；重复注册 CONFLICT、未知 NOT_FOUND。
2. 内建验证器覆盖工件存在性、Schema、引用完整性、预算约束，返回结构化 Findings 与 Confidence。
3. 受限工具执行器在调用处理器前完成 Policy 检查：工具白名单、权限、风险审批门、路径隔离、网络域白名单；越权返回 PERMISSION_DENIED。
4. HarnessRuntime 执行循环完整：策略选动作→验证→执行→事件→状态转换→工件→计量；TransitionRun 受版本守卫。
5. B0-B3 基线策略可独立运行并产出可对比轨迹（调用次数/成本/验证结果）；B4 BVAR 与 B0-B3 共享运行时、预算、审计接口。
6. 预算停止覆盖 MaxModelCalls/MaxToolCalls/MaxRuntimeSeconds/MaxCostUSD、context 取消与验证早停；停止原因可审计。
7. 定向恢复按 RecoveryLabel 分类导向 retry/repair/human_review/stop。
8. `internal/harness/` 保持领域无关；论文/PDF/DOI 等领域能力仍在 Skill 层（task7）。
9. `CGO_ENABLED=0 go build` 与 `go vet` 通过；全部测试通过。

## 设计边界

- **B0-B4 定义**（计算分配策略公平基线，源自 PROJECT_GOALS RQ2/RQ3 与归档 task8/IDEA）：
  - B0 固定单流程：单次执行，无自由路由、无修复（最低基线）。
  - B1 单 Agent 闭环：验证失败可一次定向修复（AGI 缺口修复限制在可验证槽位）。
  - B2 串行角色复用：路由/规划/执行/审查按固定顺序串行复用同一模型（D-002 单卡约束）。
  - B3 固定分支候选：候选数与选择规则固定（固定多候选或固定多 Agent，无动态触发）。
  - B4 BVAR：预算感知验证器路由（D-004：DRA→AAR→AGI→verifier→BVAR），依据风险、验证置信度、候选分歧与剩余预算动态选择动作。
- **职责分层**：验证器、工具治理、运行时、策略、预算、恢复属于 `internal/harness/`（领域无关核心）；计量接入口（MeterRecorder）由调用方注入，harness 不直接依赖 workbench。
- **工具治理**：旧 task6 工具治理（权限/风险/路径/预条件/审批门/回滚）并入本任务 task6.2，作为运行时前置检查。
- **经验与 Skill Card（D-005）**：B4 轨迹编译为 skill_card 与经验准入属于本任务后的扩展点；本任务只产出可回放轨迹，经验库逻辑留给后续任务（或在 task6.5 预留 ExperienceSink 接口）。
- **审批门**：RiskHigh/RiskCritical 动作在任务创建时即列入 Policy.ApprovalRequiredFor；运行时遇到需审批动作返回 waiting 状态并写事件，人工批准后继续（task5 已提供草案审批流）。
- **B0-B4 共享接口**：所有策略实现同一 Strategy 接口（SelectNext(ctx, state, budget, validations) → *Decision），保证对照公平。
- **领域无关回归**：至少一个模拟工具（如 file_organizer 的模拟移动/复制）在 task6 中作为工具执行器回归样例，验证核心不依赖领域对象（task7 再正式落地 Skill）。

## 子任务依赖

- task6.0 依赖 task1（ValidatorResult/Storage 契约）。
- task6.1 依赖 task6.0（Registry）+ task1（Artifact/Storage）。
- task6.2 依赖 task1（Policy/Permission/RiskLevel/ActionContract）。
- task6.3 依赖 task6.0-task6.2 + task1（RunState/TransitionRun）+ task2（Provider Chat）+ task4（MeterRecorder 接口注入）。
- task6.4 依赖 task6.3（执行循环）。
- task6.5 依赖 task6.4（B0-B3 对照）+ task6.0（验证置信度）+ task1（Budget）。
- task6.6 依赖 task6.3（执行循环内预算检查）。
- task6.7 依赖 task6.3（ErrorRecord 采集）+ task6.6（预算停止）。
- task6.8 依赖 task6.0-task6.7 全部完成。
- task6.9 依赖 task6.8（测试通过后做最终检查）。
- task6.1 与 task6.2 可部分并行；task6.4 各 B 策略实现可并行。

## 影响范围

- Affected specs: task5（draft approve→CreateRun 后执行循环由 task6 提供）、task7（Skill adapter 复用工具执行器与验证器）、task9（发布验证含 B0-B4 基线通过）。
- Affected code: `internal/harness/`（新增 validators.go、tools.go、runtime.go、strategies.go、budget.go、recovery.go）。
- 依赖 task1（契约/存储）、task2（Provider）、task4（MeterRecorder 注入）。
- 不影响 `internal/providers/`、`internal/workbench/` 现有代码（只读引用或接口注入）。

## 验证证据

- `internal/harness/errs/` 错误下沉子包 + `errors.go` alias：解除 harness ↔ providers 导入循环；providers 生产代码与测试切换至 errs。
- 新增文件：`validators.go`、`tools.go`、`runtime.go`、`strategies.go`、`budget.go`、`recovery.go`、`errs/errs.go`。
- 新增测试：`validators_test.go`（注册/冲突/NOT_FOUND/失败优先/4 内建验证器/聚合）、`tools_test.go`（白名单/权限/路径逃逸/网络域/审批门/模拟 fs/并发冒烟）、`budget_test.go`（四类上限/停止终态/计数器）、`recovery_test.go`（stop/retry→human_review/repair/unrecoverable/标签推断）、`strategies_test.go`（B0-B3 轨迹 + B4 预算感知）、`runtime_test.go`（B0 完成/计量接入/预算耗尽/Provider 失败/审批等待/取消/工具执行/验证落库），共 42 个新测试（harness 包合计 63 个）。
- 验证命令：`go build ./...`、`go vet ./...`、`go test ./...`、`CGO_ENABLED=0 go build ./...` 全部通过（cmd/server、harness、providers、workbench 全绿）。
- 说明：Windows 下 `go test -race` 需要 cgo，与 CGO_ENABLED=0 约束冲突，故未启用竞态检测；并发安全通过 RWMutex 设计 + 并发冒烟测试覆盖。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task6 spec；拆分 10 个子任务 task6.0-task6.9；明确 B0-B4 定义（固定单流程/单 Agent 闭环/串行角色复用/固定分支候选/BVAR）、工具治理并入 task6.2、计量接入点接口注入、经验库留作扩展点 |
| v0.2 | 2026-08-01 | task6.0-task6.9 全部完成并标记 completed；补齐验证证据与设计实现差异（errs 子包、停止原因可审计化、B3/B4 验证失败走 Recover 分支、-race 受 cgo 约束的说明） |
