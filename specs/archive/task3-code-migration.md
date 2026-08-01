# task3：代码迁移

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：保留现有文献搜索原型能力，将其从通用核心中拆入 Skill 层。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task3.0 | 记录原型行为 | completed | 现有来源、归档和导出有回归夹具 |
| task3.1 | 建立 `harness/` 包 | completed | 核心包可独立导入 |
| task3.2 | 拆离领域 Schema | completed | 核心无论文领域类型 |
| task3.3 | 拆离领域管线 | completed | 工作流位于 `skills/literature_search` |
| task3.4 | 建立兼容入口 | completed | 原有数据可只读导入或有明确迁移说明 |
| task3.5 | 验证工件保留 | completed | 既有导出和数据目录不被删除 |

**验证**：`tests/test_literature_skill.py` 覆盖候选、归档、引用和六项导出；`skills/literature_search/` 已拥有独立模型、来源、校验和导出实现。`rg "from bvar|import bvar" harness skills workbench tests` 无活动运行时依赖；`bvar/` 与既有数据目录保留为只读迁移参照。
