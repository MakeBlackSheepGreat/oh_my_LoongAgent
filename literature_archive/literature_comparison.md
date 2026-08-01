# 8B 与小模型多 Agent 文献对比表

检索与归档日期：2026-07-30。该表覆盖 28 篇高相关论文，聚焦“8B 或小模型、多 Agent、日常任务、工具调用、路由、基准、可靠性与 Agent 蒸馏”。

| ID | 论文 | 年份 / 载体 | 关注点 | 小模型或 8B 证据 | 多 Agent 形态 | 任务与评测 | 与本项目的关系 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| P01 | Large Language Model based Multi-Agents: A Survey of Progress and Challenges | 2024, arXiv | LLM 多 Agent 研究全景 | 无特定 8B 实验 | 架构、协作、挑战综述 | 覆盖多类 MAS | 理论背景基线 |
| P02 | Small Language Models are the Future of Agentic AI | 2025, arXiv | SLM 在 Agent 系统中的经济性与 LLM-to-SLM 转换 | 将 10B 以下作为 SLM 讨论范围 | 异构模型协同与替换策略 | 立场论文 | 与“小模型基础执行层”论点最接近 |
| P03 | A Survey on Collaborative Mechanisms Between Large and Small Language Models | 2025, arXiv | 大小模型协作机制 | 聚焦 LLM/SLM 分工 | 路由、辅助、蒸馏、融合 | 综述 | 对应模型路由与大小模型协同 |
| P04 | MARCO: Multi-Agent Real-time Chat Orchestration | 2024, EMNLP Industry | 实时多 Agent 编排 | 评测 llama-3-8b-instruct 与 Mistral-7B | 管理者编排多轮任务、工具与恢复 | 餐饮、零售等对话任务；准确率与时延 | 8B 加真实任务编排的直接先例 |
| P05 | The Fellowship of the LLMs | 2024, arXiv | 多 Agent 生成偏好优化数据 | Llama-3.1-8B、Gemma-2-9B | 生成者与审查者反馈闭环 | 合成偏好数据质量 | 可借鉴执行者与审查者分工 |
| P06 | Scaling Small Agents Through Strategy Auctions | 2026, ICML | 小 Agent 任务分配与扩展 | 研究小 Agent 的协作扩展问题 | 策略拍卖式分工 | 复杂任务与成本 | 支持“先路由、再分工”的系统设计 |
| P07 | Rethinking the Value of Multi-Agent Workflow: A Strong Single Agent Baseline | 2026, arXiv | 检验同质 MAS 是否真有额外价值 | 不以特定 8B 为主线 | 单 Agent 多轮模拟同质 MAS | 7 个基准，含规划和工具使用 | 必须纳入单 Agent 强基线 |
| P08 | Rethinking Scale: Deployment Trade-offs of Small Language Models under Agent Paradigms | 2026, ACL Industry / arXiv | <10B SLM 的 Agent 部署权衡 | 27 个开源 SLM，均小于 10B | 基础模型、工具单 Agent、协作 MAS | 20 金融数据集、8 类任务，含资源效率 | 与项目的实验设计最接近 |
| P09 | Auto-SLURP: A Benchmark for Evaluating Multi-Agent Frameworks in Smart Personal Assistant | 2025, Findings of EMNLP | 智能个人助手多 Agent 基准 | Llama-3 8B 微调用于意图模块；其余 Agent 采用 GPT-4 | 管理者协调语言理解、服务调用和响应生成 | 日程、地点、URL 等个人助理任务 | “日常”任务定义与基准设计的重要先例 |
| P10 | RiskAgent: Synergizing Language Models with Validated Tools for Evidence-Based Risk Prediction | 2025/2026, arXiv | 专家工具驱动的 Agent 系统 | 提供 RiskAgent-8B，骨干为 LLaMA-3-8B | 决策、执行、审查角色与工具协同 | 387 个医学风险场景 | 8B、工具、审查角色的端到端范例 |
| P11 | LLMSR@XLLM25: Less is More: Enhancing Structured Multi-Agent Reasoning via Quality-Guided Distillation | 2025, XLLM Workshop | 多 Agent 结构化推理与蒸馏 | 各模块基于 Meta-Llama-3-8B-Instruct 微调 | 多角色推理与质量引导 | 结构化推理评测 | 全流程 8B 多 Agent 的直接案例 |
| P12 | Can Small Agents Collaborate to Beat a Single Large Language Model? | 2026, ICLR MALGAI Workshop | 小 Agent 协作与单大模型比较 | 小模型协作实验 | 受控协作策略 | 工具密集型基准 | 用于界定何时需要多 Agent |
| P13 | A Survey on LLM-based Multi-Agent System: Recent Advances and New Frontiers in Application | 2024, arXiv | 应用、工作流和基础设施综述 | 无特定 8B 实验 | MAS 工作流与应用分类 | 多领域综述 | 补足 P01 的应用视角 |
| P14 | Planner Matters! An Efficient and Unbalanced Multi-agent Collaboration Framework for Long-horizon Planning | 2026, arXiv | 长程 GUI 任务 | Qwen2.5-VL-7B | 规划者主导的非均衡协作 | GUI 任务 | 7B 长程规划参考 |
| P15 | MERIT: Multi-Agent Collaboration for Unsupervised Time Series Representation Learning | 2025, Findings ACL | 时间序列协作学习 | 多模型协作 | 角色协作学习 | 时间序列任务 | 跨任务协作方法参考 |
| P16 | Beyond Monolithic Architectures: A Multi-Agent Search and Knowledge Optimization Framework for Agentic Search | 2026, arXiv | Agentic Search | Qwen2.5-7B | 检索、知识优化与多 Agent 搜索 | 知识密集搜索 | RAG 与搜索编排参考 |
| P17 | When Does Multi-Agent Collaboration Help? An Entropy Perspective | 2026, arXiv | 协作触发条件 | Qwen3-8B | 基于任务不确定性的协作触发 | 工具使用与复杂任务 | 何时启用多 Agent 的核心参考 |
| P18 | AMAS: Adaptively Determining Communication Topology for LLM-based Multi-agent System | 2025, EMNLP Industry | 通信拓扑 | 多模型配置 | 自适应通信拓扑 | 多 Agent 协作 | 通信成本控制参考 |
| P19 | Orchestrator Multi-Agent Clinical Decision Support System for Secondary Headache Diagnosis in Primary Care | 2025, arXiv | 临床工作流 | Llama 3.1 8B 等 | 临床角色与工具协同 | 基层医疗决策支持 | 垂直领域端到端部署参考 |
| P20 | AgentLeak: A Benchmark for Internal-Channel Privacy Leakage in Multi-Agent LLM Systems | 2026, IEEE Access | 内部通信隐私 | 多模型配置 | 多 Agent 内部通信 | 隐私泄露基准 | 安全评测指标参考 |
| P21 | CONSENSAGENT: Towards Efficient and Effective Consensus in Multi-Agent LLM Interactions through Sycophancy Mitigation | 2025, Findings ACL | 协作可靠性 | Llama 系列模型 | 共识与奉承抑制 | 多 Agent 交互 | 群体偏差治理参考 |
| P22 | MAPoRL2: Multi-Agent Post-Co-Training for Collaborative Large Language Models with Reinforcement Learning | 2025, ACL | 多 Agent 训练 | 3B 与 8B 模型 | 同伴驱动强化学习 | 通用推理任务 | 8B Agent 训练策略参考 |
| P23 | Student-Centered Distillation Narrows the Agentic Gap Between Small and Large LLMs | 2025, arXiv | 小模型 Agent 蒸馏 | 7B 学生、72B 教师 | 学生轨迹与最早错误点修正 | 12 个 Agent 基准 | 强教师到小 Agent 的闭环参考 |
| P24 | MAD-OPD: Breaking the Ceiling in On-Policy Distillation via Multi-Agent Debate | 2026, arXiv | 多教师在线策略蒸馏 | 1.7B-14B 学生、8B-32B 教师 | 教师辩论 + OPD | 5 个 Agent 与代码基准 | 多教师监督与 8B 教师参考 |
| P25 | Chain-of-Agents: End-to-End Agent Foundation Models via Multi-Agent Distillation and Agentic RL | 2025, arXiv | 多 Agent 系统蒸馏 | 多 Agent 系统蒸馏到 AFM | 协作轨迹蒸馏 + Agentic RL | Web 与 Code Agent | MAS 能力压缩与后训练参考 |
| P26 | AgentDistill: Training-Free Agent Distillation with Generalizable MCP Boxes | 2025, arXiv | 训练外 Agent 蒸馏 | 小模型学生、GPT-4o 系统对照 | 复用教师生成的 MCP 模块 | 生物医学与数学基准 | 可复用技能蒸馏参考 |
| P27 | Distilling LLM Agent into Small Models with Retrieval and Code Tools | 2025, NeurIPS Spotlight | 工具轨迹蒸馏 | 0.5B、1.5B、3B 学生 | 教师轨迹、检索/代码工具、自一致动作 | 8 个推理任务 | 工具调用蒸馏的直接对照 |
| P28 | R2V Agent: Teaching SLMs When to Ask for Help | 2026, arXiv | 强弱模型动态路由 | 4 个 SLM 骨干与强教师 | 逐步风险估计与条件升级 | HumanEval+、TextWorld、TerminalBench | 条件调用强教师的参考 |

## 当前可验证判断

1. “8B + 多 Agent + 工具调用”已有直接实现案例，P04、P10、P11 是核心证据。
2. “日常个人助手 + 多 Agent”已有专门基准，P09 是最贴近的工作。
3. “小模型采用单 Agent 还是多 Agent”已有直接比较，P07、P08、P12 应成为实验设计的必读文献。
4. 项目在提出创新点前，需要与 P08 和 P09 逐项对照模型配置、任务集、工具、基线、指标和部署环境。

## 本地文件

- PDF：`papers/`
- 下载状态、SHA-256 与原始 URL：`download_manifest.json`、`download_manifest.csv`
- 可复跑脚本：`download_all.ps1`

## 校验记录

- 归档 PDF 数：28；均已下载成功。
- 总大小：65.84 MiB。
- 已校验每个文件的 PDF 文件头与 SHA-256；`download_manifest.json` 保存对应哈希。
- 已提取每份 PDF 首页文本核对标题和页数；P09、P11 的标题与模型配置已按原文修正。
