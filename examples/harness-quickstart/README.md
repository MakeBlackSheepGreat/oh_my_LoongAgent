# Agent Harness 快速启动示例

本目录是一个**脱敏示例模板**，展示了如何为一个 Agent Harness 项目组织核心指导文件。

## 项目结构

```
harness-quickstart/
├── README.md                        # 本文件 - 项目说明
├── AGENTS.example.md                # Agent 核心指引与阅读路由
├── SOUL.example.md                  # 产品气质与代码审美
├── RULES.example.md                 # 硬边界与禁止事项
├── PROJECT_GOALS.example.md         # 项目目标与研究问题
├── SPEC.example.md                  # 计划规范与主任务表
├── IDEA.example.md                  # 候选创新与实验方向
├── ATTRIBUTIONS.example.md          # 来源与许可登记
├── specs/
│   └── README.example.md            # 任务 Spec 子目录说明
└── guidance/
    ├── README.example.md            # 决策档案目录说明
    └── DECISIONS.example.md         # 已采纳关键决策记录
```

## 文件职责

| 文件 | 职责 |
| --- | --- |
| `AGENTS.example.md` | Agent 入口、阅读路由与执行循环 |
| `SOUL.example.md` | 主观产品风格、交互气质与代码审美 |
| `RULES.example.md` | 硬边界、禁止事项和不可违反的约束 |
| `PROJECT_GOALS.example.md` | 项目目标、研究问题、指标和阶段路线 |
| `SPEC.example.md` | 计划规范、主任务表和实时状态 |
| `IDEA.example.md` | 纯候选创新、实验假设和研究方向 |
| `ATTRIBUTIONS.example.md` | 外部文献、代码、既有资产与许可的来源登记 |
| `guidance/DECISIONS.example.md` | 已采纳关键决策及其影响 |

## 使用方式

1. 将本目录复制到你的项目根目录。
2. 将 `*.example.md` 重命名为对应名称（去掉 `.example` 后缀）。
3. 根据实际项目内容替换各文件中的占位描述。
4. 在 `specs/` 中创建 `taskN-名称.md` 格式的任务说明文件。
5. 在 `guidance/DECISIONS.md` 中记录项目关键架构决策。

## 背景

此示例模板源自一个实际 Agent Harness 项目的工程实践，经脱敏处理后作为通用参考提供。所有项目特定内容（模型规格、领域描述、研究目标等）已被替换为泛化占位符。