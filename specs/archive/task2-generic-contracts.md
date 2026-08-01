# task2：通用契约

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：建立不含领域对象的任务、动作、工件、验证、预算与 Skill 契约。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task2.0 | 定义 `TaskContract` | completed | 输入、输出、权限、预算和验收条件类型化 |
| task2.1 | 定义 `ActionContract` | completed | 动作含前置条件、Schema、影响与回滚说明 |
| task2.2 | 定义 `Artifact` 与版本 | completed | 工件可寻址、哈希与溯源 |
| task2.3 | 定义 `ValidatorResult` | completed | 通过、失败、置信度和修复建议可表达 |
| task2.4 | 定义错误码与恢复标签 | completed | 失败不能只依赖自由文本 |
| task2.5 | 定义 `SkillManifest` | completed | Skill 可声明工具、验证器与导出物 |

**验证**：`tests/test_core_contracts.py` 覆盖任务、预算、动作前置条件/Schema/影响/回滚、恢复标签与 Manifest；`python -m pytest -q` 于 2026-07-31 通过。
