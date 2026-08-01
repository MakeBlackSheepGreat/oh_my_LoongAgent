# task4：状态与工件

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：为任何 Skill 提供显式、可审计、可回放的状态与工件层。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task4.0 | 实现通用状态机 | completed | created/running/waiting/terminal 转换受限 |
| task4.1 | 建立 SQLite 通用表 | completed | 运行、事件、状态、工件和验证持久化 |
| task4.2 | 实现内容寻址工件库 | completed | 内容哈希、不可变写入和路径校验有效 |
| task4.3 | 保存状态差分与依赖图 | completed | 可找出过期工件引用 |
| task4.4 | 输出 JSONL 审计 | completed | 每一步可按序读取 |
| task4.5 | 离线回放清单 | completed | 固定夹具不依赖网络重放 |

**验证**：`tests/test_core_storage.py` 覆盖状态版本、差分、父工件图、SHA-256、路径隔离、JSONL 与回放清单；`tests/test_workbench_integration.py` 覆盖文件整理 Skill 的真实存储和导出闭环。
