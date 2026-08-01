# task5：模型适配

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：以可替换、可审计的模型接口接入本地和低成本云端学生模型。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task5.0 | 定义 Provider 接口 | completed | 非流式结构化调用具备统一请求响应 |
| task5.1 | 定义结构化输出修复 | completed | 无效 JSON 可记录并定向失败 |
| task5.2 | SiliconFlow 适配 | completed | 调用与审计夹具通过 |
| task5.3 | ModelScope 适配 | completed | 调用与审计夹具通过 |
| task5.4 | 本地兼容端点适配 | completed | vLLM/llama.cpp 类端点可配置 |
| task5.5 | 预算审计 | completed | token、成本、日期、模型与限流被记录 |

**验证**：`tests/test_providers.py` 使用兼容端点夹具覆盖结构化响应、token、成本、日期、缺失凭据、限流和非法 JSON；`harness/providers.py` 通过同一 OpenAI-compatible 协议配置 SiliconFlow、ModelScope 与本地端点，密钥仅从 `HARNESS_*` 环境变量读取。
