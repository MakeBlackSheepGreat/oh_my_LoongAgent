# task9：B4 与经验

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：实现验证器驱动的 B4 策略，并把可复现轨迹转化为经验。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task9.0 | DRA 动作编译 | completed | 意图和可执行契约分离 |
| task9.1 | AAR 状态切片 | completed | 只读取依赖所需工件 |
| task9.2 | AGI 缺口修复 | completed | 单次修复限制在可验证槽位 |
| task9.3 | BVAR 路由器 | completed | 依据风险、验证和预算选择动作 |
| task9.4 | Skill Card 模型 | completed | 经验有适用条件与失效条件 |
| task9.5 | 经验编译 | completed | 只有回放通过轨迹可进入长期库 |

**验证**：`harness/experience.py` 提供 DRA、AAR、AGI 与回放约束的 Skill Card 编译；`harness/runtime.py` 以验证置信度和累计预算驱动 BVAR。`tests/test_experience.py` 与 `tests/test_runtime_service.py` 覆盖受限修复、审查与经验准入。
