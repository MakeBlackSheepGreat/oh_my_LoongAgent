# mas_medium_llms_review - Design Spec

> Human-readable design narrative — rationale, audience, style, color choices, content outline. Machine-readable execution contract: `spec_lock.md`.

## I. Project Information

| Item | Value |
| ---- | ----- |
| **Project Name** | mas_medium_llms_review |
| **Canvas Format** | PPT 16:9 (1280×720) |
| **Page Count** | 10 pages |
| **Design Style** | B) General Consulting — academic technology review |
| **Target Audience** | 本科/研究生课程教师、同专业学生、AI 应用方向听众 |
| **Use Case** | 课程论文汇报、主题综述展示、课堂演讲 |
| **Created Date** | 2026-04-28 |

---

## II. Canvas Specification

| Property | Value |
| -------- | ----- |
| **Format** | PPT 16:9 |
| **Dimensions** | 1280×720 |
| **viewBox** | `0 0 1280 720` |
| **Margins** | left/right 56px, top 48px, bottom 42px |
| **Content Area** | 1168×610, title band 92px, footer 34px |

---

## III. Visual Theme

### Theme Style

- **Style**: General Consulting, technology-academic review
- **Theme**: Light theme with navy structural accents
- **Tone**: professional, rational, clean, system-oriented, suitable for research presentation

### Color Scheme

| Role | HEX | Purpose |
| ---- | --- | ------- |
| **Background** | `#F7FAFC` | Page background |
| **Secondary bg** | `#EAF2F8` | Card and section panel background |
| **Primary** | `#0B1F3A` | Title bars, section anchors, key structure |
| **Accent** | `#1565C0` | Main highlights, icons, connectors |
| **Secondary accent** | `#22C1C3` | Layer highlights, secondary emphasis |
| **Warm accent** | `#D97706` | Risk/challenge emphasis |
| **Body text** | `#1F2937` | Main body text |
| **Secondary text** | `#64748B` | Captions and supporting labels |
| **Tertiary text** | `#94A3B8` | Footers and page numbers |
| **Border/divider** | `#CBD5E1` | Card borders and subtle dividers |
| **Success** | `#16A34A` | Positive/advantage indicators |
| **Warning** | `#D97706` | Risk and caution indicators |

### Gradient Scheme

```xml
<linearGradient id="titleGradient" x1="0%" y1="0%" x2="100%" y2="0%">
  <stop offset="0%" stop-color="#0B1F3A"/>
  <stop offset="100%" stop-color="#1565C0"/>
</linearGradient>
<radialGradient id="bgDecor" cx="88%" cy="10%" r="55%">
  <stop offset="0%" stop-color="#22C1C3" stop-opacity="0.16"/>
  <stop offset="100%" stop-color="#22C1C3" stop-opacity="0"/>
</radialGradient>
```

---

## IV. Typography System

### Font Plan

**Typography direction**: modern CJK sans, consulting-readable, PPT-safe.

| Role | Chinese | English | Fallback tail |
| ---- | ------- | ------- | ------------- |
| **Title** | `"Microsoft YaHei", "PingFang SC"` | `Arial` | `sans-serif` |
| **Body** | `"Microsoft YaHei", "PingFang SC"` | `Arial` | `sans-serif` |
| **Emphasis** | `"Microsoft YaHei"` | `Arial` | `sans-serif` |
| **Code** | — | `Consolas, "Courier New"` | `monospace` |

**Per-role font stacks**:

- Title: `"Microsoft YaHei", "PingFang SC", Arial, sans-serif`
- Body: `"Microsoft YaHei", "PingFang SC", Arial, sans-serif`
- Emphasis: `Arial, "Microsoft YaHei", sans-serif`
- Code: `Consolas, "Courier New", monospace`

### Font Size Hierarchy

**Baseline**: Body font size = 20px.

| Purpose | Ratio to body | Current project size | Weight |
| ------- | ------------- | -------------------- | ------ |
| Cover title | 3.6× | 72px | Bold |
| Chapter / section opener | 2.2× | 44px | Bold |
| Page title | 1.7× | 34px | Bold |
| Hero number / key phrase | 2.1× | 42px | Bold |
| Subtitle | 1.25× | 25px | SemiBold |
| **Body content** | **1×** | **20px** | Regular |
| Annotation / caption | 0.75× | 15px | Regular |
| Page number / footnote | 0.6× | 12px | Regular |

---

## V. Layout Principles

### Page Structure

- **Header area**: 48–122px; page title, section marker, optional small icon.
- **Content area**: 122–650px; varied between architecture diagram, role matrix, process flow, comparison cards, and roadmap.
- **Footer area**: 650–700px; source note, short takeaway, page number.

### Layout Pattern Library

| Pattern | Used For |
| ------- | -------- |
| **Single column centered** | Cover and conclusion pages |
| **Asymmetric split** | Background/context and challenge pages |
| **Three/four column cards** | Main role and future trend pages |
| **Matrix grid / comparison table** | Model role and application scenario pages |
| **Pipeline with stages** | Technology foundation page |
| **Layered architecture** | MAS architecture page |
| **Hub & spoke** | Core role of mid-small LLMs |
| **Roadmap vertical** | Future outlook page |

### Spacing Specification

**Universal**:

| Element | Recommended Range | Current Project |
| ------- | ---------------- | --------------- |
| Safe margin from canvas edge | 40-60px | 56px |
| Content block gap | 24-40px | 28px |
| Icon-text gap | 8-16px | 12px |

**Card-based layouts**:

| Element | Recommended Range | Current Project |
| ------- | ---------------- | --------------- |
| Card gap | 20-32px | 24px |
| Card padding | 20-32px | 24px |
| Card border radius | 8-16px | 14px |
| Three-column card width | 360-380px | 356-370px |

---

## VI. Icon Usage Specification

### Source

- **Built-in icon library**: `templates/icons/phosphor-duotone/`
- **Usage method**: SVG placeholder `<use data-icon="phosphor-duotone/icon-name" .../>`.
- **Deck-wide rule**: only `phosphor-duotone` for generic icons; no mixing with other stylistic libraries.

### Recommended Icon List

| Purpose | Icon Path | Page |
| ------- | --------- | ---- |
| AI agent / model | `phosphor-duotone/robot` | P01, P03 |
| Multi-agent collaboration | `phosphor-duotone/users-three` | P02, P04 |
| System flow | `phosphor-duotone/flow-arrow` | P03, P06 |
| Safety governance | `phosphor-duotone/shield-check` | P04, P08 |
| Edge/cloud deployment | `phosphor-duotone/cloud-check` | P05, P09 |
| Knowledge/RAG | `phosphor-duotone/database` | P03, P06 |
| Evaluation and metrics | `phosphor-duotone/chart-line-up` | P05, P08 |
| Risk warning | `phosphor-duotone/warning-circle` | P08 |
| Future outlook | `phosphor-duotone/rocket-launch` | P09, P10 |
| Model reasoning | `phosphor-duotone/brain` | P02, P06 |
| Architecture layers | `phosphor-duotone/tree-structure` | P03 |
| Technical foundation | `phosphor-duotone/cpu` | P06 |
| Workflow | `phosphor-duotone/gear-six` | P06 |
| Research context | `phosphor-duotone/book-open-text` | P02 |

---

## VII. Visualization Reference List

**Read-audit**:

```text
Catalog read: 70 templates / 10 categories
Per-page selection:
  P03 layered_architecture | summary-quote: "Pick for 3-4 horizontal architecture layers (e.g. presentation/service/data), 2-4 module cards per layer."
  P04 hub_spoke            | summary-quote: "Pick for 1 core capability + 4-8 surrounding capabilities (platform/ecosystem)."
  P06 pipeline_with_stages | summary-quote: "Pick for 3-5 stage horizontal pipeline where each stage = title + 1-line description + output artifact, connected by directional arrows."
  P07 comparison_table     | summary-quote: "Pick for 2-4 plans/products compared across many feature rows (dense matrix)."
  P09 roadmap_vertical     | summary-quote: "Pick for 4-8 milestones on a vertical timeline with status indicators."
Runners-up considered:
  vertical_list rejected for P04: the page centers one core role with surrounding capabilities, so hub_spoke fits better.
  process_flow rejected for P06: the technology page needs named outputs per stage, so pipeline_with_stages is more specific.
  comparison_columns rejected for P07: application scenarios require dense matrix rows, not marketing tier cards.
```

| Visualization Type | Reference Template | Used In |
| ------------------ | ------------------ | ------- |
| layered_architecture | `templates/charts/layered_architecture.svg` | Slide 03 |
| hub_spoke | `templates/charts/hub_spoke.svg` | Slide 04 |
| pipeline_with_stages | `templates/charts/pipeline_with_stages.svg` | Slide 06 |
| comparison_table | `templates/charts/comparison_table.svg` | Slide 07 |
| roadmap_vertical | `templates/charts/roadmap_vertical.svg` | Slide 09 |

---

## VIII. Image Resource List

No external images or AI-generated images are used. The deck relies on native SVG shapes, icons, diagrams, cards, and tables.

---

## IX. Content Outline

### Part 1: Problem framing

#### Slide 01 - Cover

- **Layout**: Single column centered with gradient header and abstract node background
- **Title**: 中小型大模型在多智能体系统中的作用与发展前景综述
- **Subtitle**: 低成本、可部署、可角色化的基础执行层
- **Content**: course report / AI systems review / 2026
- **Rhythm**: anchor

#### Slide 02 - Why this topic matters

- **Layout**: Asymmetric split, left narrative, right three evidence cards
- **Title**: 从单模型问答到多智能体协作
- **Content**:
  - 大模型应用从“回答”走向“执行”
  - 多智能体系统强调角色分工、通信协作、工具调用
  - 成本、时延、权限和可靠性成为工程约束
- **Visualization**: vertical_list style custom layout
- **Rhythm**: dense

### Part 2: Concepts and architecture

#### Slide 03 - What is an LLM-based MAS?

- **Layout**: Layered architecture
- **Title**: 多智能体系统不是多个聊天框，而是复合系统
- **Content**:
  - 角色层：规划者、研究员、执行者、审查者
  - 能力层：模型能力、记忆、RAG、工具调用
  - 编排层：状态管理、工作流、升级机制
  - 治理层：权限、审计、安全控制
- **Visualization**: layered_architecture
- **Rhythm**: dense

#### Slide 04 - Core positioning of mid-small LLMs

- **Layout**: Hub & spoke
- **Title**: 中小型大模型是多智能体系统的“基础执行层”
- **Content**:
  - 低成本高频调用
  - 角色专用化
  - 私有化和边缘部署
  - 工具/RAG 节点执行
  - 安全过滤与评审
  - 与大模型形成升级协同
- **Visualization**: hub_spoke
- **Rhythm**: breathing

### Part 3: Functions and technologies

#### Slide 05 - Four practical roles

- **Layout**: 2×2 card matrix
- **Title**: 四类核心作用：成本、角色、部署、治理
- **Content**:
  - 成本与时延优化层
  - 角色专业化与协作分工
  - 私有化、边缘化与端边云协同
  - 安全控制、过滤与评审节点
- **Visualization**: custom matrix cards
- **Rhythm**: dense

#### Slide 06 - Technology foundation

- **Layout**: Pipeline with stages
- **Title**: 关键技术把“小模型”变成“可用节点”
- **Content**:
  - 知识蒸馏：角色策略迁移
  - 量化压缩：降低部署门槛
  - LoRA/QLoRA：低成本领域适配
  - 模型路由：大小模型分层调用
  - RAG/工作流：补足知识与流程边界
- **Visualization**: pipeline_with_stages
- **Rhythm**: dense

### Part 4: Applications and risks

#### Slide 07 - Application scenario matrix

- **Layout**: Dense comparison table
- **Title**: 典型场景：高频、边界清晰、需治理的任务最适配
- **Content**:
  - 企业知识助手
  - 软件工程协作
  - 客服与工单
  - 边缘设备助手
  - 垂直行业辅助
- **Visualization**: comparison_table
- **Rhythm**: dense

#### Slide 08 - Challenges and risk boundaries

- **Layout**: Two-column risk map with warning band
- **Title**: 多智能体不天然更可靠：错误可能被放大
- **Content**:
  - 能力上限：复杂推理、长上下文、高风险决策
  - 协作风险：级联幻觉、群体思维、责任模糊
  - 工具风险：越权调用、接口失败、不安全输出
  - 评估不足：不能只看单模型 benchmark
- **Visualization**: pros_cons / risk cards custom layout
- **Rhythm**: dense

### Part 5: Outlook and conclusion

#### Slide 09 - Development outlook

- **Layout**: Vertical roadmap
- **Title**: 未来方向：从模型大小竞争转向系统协同
- **Content**:
  - 大小模型协同路由
  - 角色专用中小模型
  - 端—边—云多智能体架构
  - 多智能体轨迹蒸馏
  - 安全治理内生化
- **Visualization**: roadmap_vertical
- **Rhythm**: breathing

#### Slide 10 - Conclusion

- **Layout**: Single column centered, three takeaway strips
- **Title**: 结论：真正成熟的是“模型—工具—知识—流程—安全”的协同结构
- **Content**:
  - 中小型大模型不是替代品，而是基础执行层
  - 适合高频、明确、可编排、可治理节点
  - 未来价值取决于系统级评估与安全治理成熟
- **Rhythm**: anchor

---

## X. Speaker Notes Requirements

- **Filename**: match SVG name, e.g. `01_cover.md`; master file at `notes/total.md`.
- **Total duration**: 8–10 minutes.
- **Notes style**: formal but clear, suitable for classroom presentation.
- **Presentation purpose**: inform and synthesize.
- **Each page**: 45–70 seconds, include opening sentence, key explanation, transition cue.

---

## XI. Technical Constraints Reminder

### SVG Generation Must Follow:

1. viewBox: `0 0 1280 720`
2. Background uses `<rect>` elements
3. Text wrapping uses `<tspan>`; `<foreignObject>` forbidden
4. Transparency uses `fill-opacity` / `stroke-opacity`; `rgba()` forbidden
5. Forbidden: `mask`, `<style>`, `class`, `foreignObject`, `textPath`, `animate*`, `script`, `<iframe>`
6. Text symbols use raw Unicode; XML reserved chars must be escaped.
7. Icons use placeholders from the approved `phosphor-duotone` inventory only.

### PPT Compatibility Rules:

- `<g opacity="...">` forbidden; set opacity on each child.
- Inline attributes only; no external CSS or `@font-face`.
- Export from `svg_final/` after `finalize_svg.py`, never directly from raw SVG.