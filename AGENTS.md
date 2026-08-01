# 项目 Agent 核心指引

你是本项目的实现与研究 Agent。持续推动“面向 12B 以下模型的日常多 Agent 系统”形成可用、可回放、可治理、可复现实验的真实工程。

## 必读顺序

1. `SOUL.md`：理解产品气质、交互风格与代码口味。
2. `RULES.md`：确认硬边界、禁止事项与技术约束。
3. `PROJECT_GOALS.md`：确认长期目标、研究问题和验收指标。
4. `SPEC.md`：确认当前阶段、任务拆分和计划状态。
5. `specs/taskN-*.md`：读取当前 `in_progress` 任务的详细规格。
6. `ATTRIBUTIONS.md`：使用文献、外部代码、模型或既有资产前，确认来源、用途与许可。
7. `IDEA.md`：仅在研究、创新或实验设计相关任务中读取。
8. `guidance/DECISIONS.md`：遇到架构取舍或历史决策时读取。

发生冲突时，依次遵循：用户当前要求、`RULES.md`、当前任务 Spec、`PROJECT_GOALS.md`、`SOUL.md`、`IDEA.md`。`IDEA.md` 中的内容均为候选假设，不能自动成为产品要求。

## 工作循环

1. 定位当前 `taskN`，核对输入、输出、依赖与验收条件。
2. 大型任务先更新 `SPEC.md`，再在 `specs/` 中拆成 `taskN.0`、`taskN.1` 等小任务。
3. 一次只推进一个明确的 `in_progress` 小任务；完成后立即更新状态和验证证据。
4. 实现遵循 `SOUL.md` 的风格与 `RULES.md` 的边界；重要取舍写入 `guidance/DECISIONS.md`。
5. 使用外部论文、代码、模型、数据或既有资产时，先在 `ATTRIBUTIONS.md` 登记可验证来源、用途、适用路径与许可状态；研究结论在正文中给出对应引用。
6. 新的创新假设、可实验机制或研究方向写入 `IDEA.md`；项目要求、编码规范和进度不得写入该文件。
7. 结束前运行与风险相称的验证，并同步相关 Spec 状态。

## 项目地图

| 路径 | 唯一职责 |
| --- | --- |
| `AGENTS.md` | Agent 入口、阅读路由与执行循环 |
| `SOUL.md` | 主观产品风格、交互气质与代码审美 |
| `RULES.md` | 硬边界、禁止事项和不可违反的约束 |
| `PROJECT_GOALS.md` | 项目目标、研究问题、指标和阶段路线 |
| `SPEC.md` | 计划规范、主任务表和实时状态 |
| `specs/` | 每个大型任务的可执行拆分与验收记录 |
| `IDEA.md` | 纯候选创新、实验假设和研究方向 |
| `ATTRIBUTIONS.md` | 外部文献、代码、既有资产与许可的来源登记 |
| `LICENSE`、`NOTICE` | 项目发布许可及第三方归属声明 |
| `guidance/DECISIONS.md` | 已采纳关键决策及其影响 |
| `guidance/REFLECTIONS.md` | 用户指出的工作方式问题、根因与防再犯措施（每次被指出问题必须登记） |
| `FOUNDATION_MODEL_*.md` | 基础模型论文迁移分析 |
| `literature_archive/` | 论文、全文、对比表和检索审计 |
| `internal/harness/` | 与领域无关的 Go 契约、状态、工件、工具、验证器、Provider 与运行时 |
| `internal/workbench/` | Go 应用层：账户、项目、会话、供应商、计量与 i18n |
| `internal/providers/` | OpenAI 兼容 provider 与 DeepSeek 定制化 |
| `internal/skills/` | Skill 注册与运行时适配器 |
| `cmd/server/`、`cmd/cli/` | Go 服务入口与管理 CLI |
| `web/` | Vue 3 + TypeScript 本地工作台前端 |
| `configs/` | `harness.toml` 等 config-driven 声明 |
| `projects/`、`ppt-master/` | 既有汇报与演示资产 |
