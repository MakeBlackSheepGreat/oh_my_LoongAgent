# task1：重新冻结通用 Harness

**状态**：completed  
**更新时间**：2026-07-31

## 目标

将项目本体冻结为通用小模型 Agent Harness，并将论文搜索明确为首个 Skill Pack、首个发布场景和首个实验场景。

## 输入

- `PROJECT_GOALS.md`
- `IDEA.md`
- `FOUNDATION_MODEL_AGENT_TRANSFER_MATRIX.md`
- `FOUNDATION_MODEL_TO_AGENT_IDEAS.md`
- `literature_archive/literature_comparison.xlsx`

## 输出

- 通用核心、Skill Pack、Provider、工具、验证器、工件库和工作台的责任边界。
- `BVAR` 作为 B4 动态路由策略的固定定义。
- 论文搜索作为首个 Skill Pack；模拟文件整理作为领域无关性回归样例。
- 强教师的离线参考包边界与可观察数据规则。

## 子任务

| ID | 内容 | 状态 | 依赖 | 验收条件 |
| --- | --- | --- | --- | --- |
| task1.0 | 更新项目目标为通用 Harness | completed | 无 | `PROJECT_GOALS.md` 定义通用核心 |
| task1.1 | 固定 BVAR 为 B4 策略 | completed | task1.0 | D-009 记录策略边界 |
| task1.2 | 固定首发论文搜索 Skill | completed | task1.0 | 首发场景不定义框架边界 |
| task1.3 | 固定教师隔离原则 | completed | task1.0 | D-010 禁止教师进入交互运行时 |
| task1.4 | 固定模块边界 | completed | task1.1-task1.3 | 核心不得依赖领域对象 |
| task1.5 | 冻结首轮验收指标 | completed | task1.4 | B0-B4、工件、验证与回放标准已定义 |

## 计划变更记录

| 日期 | 变更 |
| --- | --- |
| 2026-07-30 | 从已有 P0 路线拆分出首个动态任务 Spec |
| 2026-07-31 | 将论文搜索降为首个 Skill Pack，完成通用 Harness 的边界冻结。 |
