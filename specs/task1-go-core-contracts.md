# task1：Go 核心契约与存储

**状态**：completed  
**目标**：以 Go struct + 验证标签实现领域无关的 Agent Harness 核心契约与 SQLite 存储层，覆盖状态机、版本守卫、内容寻址工件、事件、依赖图与回放。

## 输入

- D-013、D-014、C-007。
- 旧 Python 契约与存储的领域语义（已删除，从 `specs/archive/task2-generic-contracts.md` 与 `specs/archive/task4-state-artifacts.md` 的历史记录追溯）。
- Go 1.26.5、modernc.org/sqlite（纯 Go，CGO_ENABLED=0）。

## 输出

- `internal/harness/contracts.go`：全部契约类型与构造校验。
- `internal/harness/errors.go`：HarnessError 与错误码。
- `internal/harness/storage.go`：HarnessStore 完整实现。
- `internal/harness/*_test.go`：测试覆盖契约校验、状态机、版本守卫、工件、事件、回放。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task1.0 | 契约类型与构造校验 | completed | 15 个契约类型覆盖 RunStatus/Budget/Policy/TaskContract/ActionContract/Artifact/RunState/ValidatorResult/SkillManifest/Event 等；构造函数执行校验 |
| task1.1 | HarnessError 与错误码 | completed | 错误码常量与快捷构造函数对齐 Python errors.py |
| task1.2 | SQLite 存储层 | completed | CreateRun/GetRun/ListRuns/StateVersions/TransitionRun/AppendEvent/EventsAfter/PutArtifact/GetArtifact/ListArtifacts/ArtifactDependencyGraph/ReadArtifact/RecordValidatorResult/RecordError/ReplayManifest 全部实现 |
| task1.3 | 测试覆盖 | completed | 21 个测试全部通过 |
| task1.4 | CGO_ENABLED=0 编译验证 | completed | `CGO_ENABLED=0 go build` 与 `go vet` 通过 |
| task1.5 | 交叉编译验证 | completed | linux/loong64、linux/riscv64、darwin/arm64、linux/arm64、windows/arm64 全部通过 |

## 验收条件

1. Go struct 覆盖全部核心契约，验证逻辑在构造函数中执行。
2. SQLite schema 就绪（runs/state_versions/events/artifacts/validator_results/errors 表+索引）。
3. 状态机转换受版本守卫保护；非法转换返回错误。
4. 工件按 SHA-256 内容寻址；完整性校验通过。
5. 回放清单可导出为 JSON 文件。
6. `CGO_ENABLED=0 go build` 与 `go vet` 通过。
7. 全部测试通过。

## 验证证据

- `internal/harness/contracts.go`：覆盖 RunStatus、RiskLevel、RecoveryLabel、Permission、Budget、Policy、TaskContract、ActionContract、ArtifactInput、Artifact、ErrorRecord、RunState、ValidatorResult、SkillManifest、Event 全部契约。
- `internal/harness/errors.go`：HarnessError 与错误码常量（VALIDATION_ERROR/NOT_FOUND/BUDGET_EXCEEDED/PERMISSION_DENIED/CONFLICT/PROVIDER_UNAVAILABLE/INTERNAL_ERROR）。
- `internal/harness/storage.go`：HarnessStore 完整实现，含状态机转换表、版本守卫、内容寻址工件（SHA-256）、事件 JSONL 追加、依赖图、回放清单。
- SQLite schema：runs/state_versions/events/artifacts/validator_results/errors 六张表+索引。
- 依赖：modernc.org/sqlite（纯 Go，支持 CGO_ENABLED=0）。
- 测试：21 个 Go 测试全部通过（`go test ./internal/harness/ -v -count=1`）。
- 编译：`CGO_ENABLED=0 go build` 与 `go vet` 均通过。
- 交叉编译：linux/loong64、linux/riscv64、darwin/arm64、linux/arm64、windows/arm64 全部通过。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task1，继承原 task14.0+14.1 成果；契约、错误、存储与测试全部就绪 |
