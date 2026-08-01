# task7：验证器

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：让确定性验证结果成为路由、修复和停止决策的依据。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task7.0 | 验证器注册表 | completed | Skill 可声明并调用验证器 |
| task7.1 | Schema 验证器 | completed | 输入输出契约可独立检查 |
| task7.2 | 工件差分验证器 | completed | 修改范围和依赖可检测 |
| task7.3 | 规则与测试验证器 | completed | 可返回结构化失败和证据 |
| task7.4 | 证据验证器接口 | completed | 文献引用验证为领域插件 |
| task7.5 | 置信度和失败聚类 | completed | BVAR 可消费验证特征 |

**验证**：`harness/validators.py` 提供注册表、JSON Schema、工件依赖和失败聚类；`skills/literature_search/validators.py` 是领域证据插件。`tests/test_core_tools.py` 与 `tests/test_literature_skill.py` 证明检查结果来自工件与来源观察。
