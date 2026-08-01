# 来源与许可登记

**版本**：v0.2  
**更新时间**：2026-08-01

本文件是项目对外部文献、代码、既有资产、模型报告与许可状态的唯一登记处。每条研究主张、复用实现、分发材料和发布包都应能从这里追溯到来源、用途、版本与权利状态。

## 使用规则

1. 引用论文时，在输出正文、图表或实验设计旁给出作者、年份、标题和持久标识符；本文件保存完整归档位置与核验信息。
2. 使用外部代码前，登记上游 URL、版本或 commit、许可证、使用路径、改动说明与保留声明。仅有模糊的设计印象不足以支持代码来源记录。
3. 新增来源先补充登记，再进入代码、任务集、报告或发布包。许可信息缺失时，将条目标记为 `blocked`。
4. 每次发布前复核本文件、`LICENSE`、`NOTICE` 和实际依赖清单。

## 文献登记

### L-001：多 Agent 与小模型协作语料

- **范围**：28 篇下载论文，标识为 `P01` 至 `P28`。
- **登记位置**：`literature_archive/download_manifest.csv`。
- **每篇已有字段**：论文 ID、归档文件名、原始 URL、下载状态、字节数与 SHA-256。
- **完整性**：截至 2026-07-30，28/28 条状态为 `downloaded`。
- **使用范围**：多 Agent 协作、SLM/LLM 协同、工作流编排、风险治理、教师反馈与 Agent 蒸馏的相关论证。
- **引用要求**：研究正文不得只引用 `Pxx` 编号；应以论文的作者、年份、标题和 DOI、arXiv 或 ACL Anthology 标识符呈现，并将该标识符链接到归档条目。

### L-002：基础模型到 Agent 迁移语料

- **范围**：59 篇基础模型论文与技术报告，按 arXiv ID 唯一标识。
- **登记位置**：`literature_archive/foundation_model_reading/manifest.csv`。
- **每篇已有字段**：厂商、arXiv ID、标题、页数、解析状态与全文文本路径。
- **完整性**：截至 2026-07-30，59/59 条状态为 `extracted`。
- **使用范围**：长上下文、稀疏注意力、MoE、推理时计算、强化学习、验证反馈、模型服务与 Agent Harness 的迁移分析。
- **引用要求**：`FOUNDATION_MODEL_AGENT_TRANSFER_MATRIX.md` 和 `FOUNDATION_MODEL_TO_AGENT_IDEAS.md` 中的论断在进入论文、实验报告或产品文档时，必须标注对应 arXiv ID 与原始标题。

### L-003：项目内文献分析材料

- **范围**：`literature_archive/literature_comparison.xlsx`、`literature_archive/literature_comparison.md`、`FOUNDATION_MODEL_AGENT_TRANSFER_MATRIX.md` 与 `FOUNDATION_MODEL_TO_AGENT_IDEAS.md`。
- **状态**：派生研究材料。
- **来源要求**：它们用于组织、比较和提出实验假设；每项可验证学术主张仍需回链至 L-001 或 L-002 的原始论文。

## 代码与资产登记

### C-001：用户既有项目资产

- **来源**：GitHub 账户 `MakeBlackSheepGreat`，`https://github.com/MakeBlackSheepGreat`；本次偏好校准审查了 `AI-Novel-APP`、`bucad`、`ct-nodule-risk`、`swust-code`、`oh_my_LoongAgent`、`IPC_vision`、`pdf-vector-search`、`html-png-exporter-skill` 与 `PCB-Agent-Teams-JLCEDA`，以及本地维护的 `osteo-vision` 项目。
- **当前使用**：为 `SOUL.md` 提供工程与交互偏好证据；当前核心项目未复制其中的源代码、依赖清单、提示词、配置或测试夹具。
- **复用前提**：复用时新增具体仓库 URL、commit、文件路径、原许可证、改动说明和归属文本；fork 项目需同时登记上游来源与用户修改范围。
- **状态**：`registered_for_inspiration_only`。

### C-002：`ppt-master/` 目录

- **来源 URL**：`https://github.com/hugohe3/ppt-master`。
- **版本与许可**：目录内 README 标注 v2.3.0；`ppt-master/LICENSE` 声明为 MIT License，Copyright (c) 2025-2026 Hugo He。
- **当前使用**：既有演示资产；不属于本项目核心运行时、任务集或评测基线。
- **发布条件**：在单独发布、修改或将其接入核心系统前，固定上游 commit，复核完整依赖许可并补充具体使用范围。
- **状态**：`registered_no_core_use`。

### C-003：当前项目原创内容

- **范围**：根目录项目指导文件、研究记录，以及后续由项目作者创建且未复用外部受版权保护代码的实现。
- **许可证**：Apache License 2.0，见根目录 `LICENSE`。
- **例外**：含有独立上游许可证或在 C-001、C-002 或后续条目中登记的内容，依其上游许可和归属要求处理。

### C-004：核心运行时依赖（Python，已退役）

- **来源 URL**：FastAPI (`https://github.com/fastapi/fastapi`)、HTTPX (`https://github.com/encode/httpx`)、Pydantic (`https://github.com/pydantic/pydantic`)、openpyxl (`https://foss.heptapod.net/openpyxl/openpyxl`)、pypdf (`https://github.com/py-pdf/pypdf`) 与 Uvicorn (`https://github.com/encode/uvicorn`)。
- **版本约束**：见 `pyproject.toml`；发布前必须将实际锁定版本写入复现材料。
- **许可证**：FastAPI、Pydantic、openpyxl 为 MIT；HTTPX、pypdf、Uvicorn 为 BSD-3-Clause。以发布时各依赖发行包携带的许可证为准。
- **项目内路径**：`harness/`、`skills/`、`workbench/` 和测试环境。
- **使用范围**：Web API、兼容模型 HTTP 调用、Schema、Excel、PDF、ASGI 服务与测试。
- **改动与归属要求**：只通过包管理器使用，不复制上游源文件；发布审计保留每个依赖的适用许可证与版权声明。
- **状态**：retired（D-014 决定删除 Python 后端，这些依赖不再使用）。

### C-005：CC Switch 供应商管理参考

- **来源 URL**：https://github.com/farion1231/cc-switch 与 https://ccswitch.io/ 。
- **作者或维护者**：Jason Young / CC Switch contributors。
- **版本或 commit**：`main`，2026-07-31 核验 commit `4bfb3fc30d78ecfbc41340da3ad72720ee28843c`。
- **许可证或使用条款**：上游仓库为 MIT License；产品网站内容遵循其公开使用条款。
- **项目内路径**：原 `workbench/workspace.py`、`harness/providers.py`、`web/src/components/ProviderManager.tsx` 与相关测试（Python/React 实现已于 D-014 删除；Go 版本在 `internal/workbench/`、`internal/providers/`、`web/` 重新实现）。
- **使用范围**：产品行为参考，包括供应商档案、显式激活、环境变量配置、连通性检查、状态诊断和本地 SQLite 持久化的交互边界。
- **改动与归属要求**：不复制上游代码、图标、文案、截图、品牌元素或配置文件；本项目原创实现遵循 Apache-2.0，发布资料保留本条来源说明。
- **状态**：registered_for_design_inspiration_only（设计参考，实现已迁移到 Go/Vue）。

### C-006：Codex 与 Claude Code 工作台交互参考

- **来源 URL**：https://developers.openai.com/codex/ 与 https://code.claude.com/docs/en/overview 。
- **作者或维护者**：OpenAI；Anthropic。
- **版本、日期或标识**：2026-07-31 访问的公开产品与文档页面。
- **许可证或使用条款**：产品与文档受各权利人使用条款保护。
- **项目内路径**：原 `web/src/components/ProjectSidebar.tsx`、`web/src/components/ChatWorkspace.tsx`、`web/src/components/Inspector.tsx` 与 `web/src/styles.css`（React 实现已于 D-014 删除；Vue 版本在 `web/` 重新实现）。
- **使用范围**：项目上下文、连续任务、聊天内阶段反馈、计划和工具证据渐进披露的交互原则。
- **改动与归属要求**：不复制代码、品牌、图标、视觉资产或受版权保护文案；只采用抽象的产品交互原则。
- **状态**：registered_for_design_inspiration_only（设计参考，实现已迁移到 Vue）。

### C-007：DeepSeek-Reasonix Go 架构参考

- **来源 URL**：https://github.com/esengine/DeepSeek-Reasonix
- **作者或维护者**：esengine / SivanCola 及 Reasonix contributors。
- **版本或 commit**：`main-v2` 分支，2026-07-31 核验 commit `d534de0fa2ab552099e863a80458074b13e4348c`。
- **许可证或使用条款**：MIT License。
- **项目内路径**：Go 重写后的 `internal/` 核心与 `cmd/` 入口（规划中）；当前无源代码复制。
- **使用范围**：Go 单二进制 agent harness 的架构模式参考，包括 config-driven provider 声明（`reasonix.toml`）、MCP 兼容 plugin 体系（stdio JSON-RPC）、cache-aware context maintenance（stale 输出剪枝、prefix cache 稳定会话）、`CGO_ENABLED=0` 跨平台编译与单二进制分发。
- **改动与归属要求**：不复制上游源代码、品牌、图标、文案或配置文件；本项目参考其架构模式原创实现，遵循 Apache-2.0，发布资料保留本条来源说明与 MIT 许可归属。
- **状态**：registered_for_architecture_inspiration_only。

## 媒体与传闻登记

下列条目登记公开媒体报道中的产品信息、团队动态与事件性信息。它们未经官方文档、论文、代码仓库或正式公告的直接核验，仅用于设计参考与背景判断；不得在研究报告正文中作为事实或学术引用。

### R-001：DeepSeek Harness 内测与团队背景报道

- **事件**：2026-07 下旬，多家中媒报道 DeepSeek Harness 内部测试招募截图在社区流传、Harness 团队于 2026-03 由崔添翼（浙江大学计算机系，前 Jane Street 量化交易系统工程师约 9 年）组建，并基于团队招聘信息推断产品方向：KV 前缀缓存与长上下文压缩、向量记忆与会话持久化、工具链式调用与错误回退、多 Agent 通信协议、任务规划图自优化、动态工具注册。
- **来源报道（示例）**：
  - 头条号《DeepSeek Harness开启内测?看来V4正式版也不远了》http://m.toutiao.com/group/7667886878187602472/ （2026-07）
  - 头条号《DeepSeekHarness内测开启，V4正式版近了?》http://m.toutiao.com/group/7668372407379444258/ （2026-07）
  - 搜狐《DeepSeek Harness开启内测，V4正式版或将同步发布》https://m.sohu.com/a/1056397054_122066678/ （2026-07）
  - 搜狐《DeepSeek Harness内测将启，V4正式版或携三大升级同步登场》https://m.sohu.com/a/1056458233_122066678/ （2026-07）
- **核验状态**：截图原件与官方公告未获确认；上述均为二手转述，产品形态为媒体报道推断，未经官方确认。
- **使用范围**：IDEA.md I-018 的依据来源。
- **状态**：registered_for_design_inspiration_only（媒体推断，未经官方确认）。

### R-002：《Thinking with Visual Primitives》发布后撤回事件报道

- **事件**：2026-04-30，DeepSeek 多模态团队（陈小康）发布 25 页技术论文《Thinking with Visual Primitives》（以视觉原语思考，与北京大学、清华大学合作），提出以边界框和点作为最小思考单元、模型"边指边想"的视觉原语推理，以及 patch 空间压缩与稀疏注意 KV 压缩的两级压缩；05-01 论文与相关仓库被撤回，官方未作解释。完整内容仅存于第三方转载（如 CSDN 转载的 APPSO 留存副本）。
- **来源报道（示例）**：
  - CSDN《DeepSeek光速撤回的神秘论文讲了什么?》https://blog.csdn.net/qq_46366530/article/details/160739144 （2026-05-03）
  - 头条号《DeepSeek被删掉的论文到底有多可怕?》http://m.toutiao.com/group/7635854833597104675/ （2026-05）
  - 头条号《DeepSeek"看见"了!视觉论文却蹊跷被撤，陈小康发了什么?》http://m.toutiao.com/group/7637302290767938098/ （2026-05）
- **核验状态**：论文无官方存档（GitHub 404、arXiv 无记录确认），内容仅二手转述；撤回原因官方未解释，报道猜测不作为事实采信。
- **使用范围**：背景与设计参考，当前未进入 IDEA 条目。
- **状态**：registered_for_background_only（二手转述，未经官方确认）。

## 发布检查表

- 每个研究结论都含有定位到 L-001 或 L-002 的原始论文引用。
- 每个外部代码条目都具备 URL、版本或 commit、许可证、文件路径与使用说明。
- `LICENSE` 与 `NOTICE` 覆盖发布物；各依赖和第三方目录保留其原始许可。
- 论文 PDF、模型权重、数据集、商标和外部图像均保留原权利人的适用权利，不因本项目归档而获得再许可。

## 新条目模板

```md
### L-xxx 或 C-xxx：名称

- 来源 URL：
- 作者或维护者：
- 版本、DOI、arXiv ID 或 commit：
- 许可证或使用条款：
- 项目内路径：
- 使用范围：
- 改动与归属要求：
- 状态：registered | blocked | retired
```
