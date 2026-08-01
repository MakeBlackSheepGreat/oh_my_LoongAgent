# task10：论文搜索 Skill

**状态**：retired  

> **迁移说明（D-014，2026-08-01）**：本任务的 Python 实现已删除，不做存档。相关能力以 Go 在 task14 对应子任务中重新实现。下方验证证据为历史记录，仅供追溯。

**目标**：通过通用 Harness 交付可追溯的论文搜索证据包。

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task10.0 | Skill Manifest 和任务输入 | completed | 研究问题、来源、预算和导出要求可验证 |
| task10.1 | 四类来源工具 | completed | arXiv/OpenAlex/Crossref/ACL 各有夹具 |
| task10.2 | 去重与来源溯源 | completed | DOI、ID、标题相似度分层合并 |
| task10.3 | 开放全文归档 | completed | MIME、大小、版权状态和哈希被保存 |
| task10.4 | 证据与引用核验 | completed | 引用只能指向候选池 |
| task10.5 | 四种导出 | completed | Markdown、Excel、JSON、归档清单可下载 |

**验证**：`tests/test_literature_sources.py` 为 arXiv、OpenAlex、Crossref、ACL 和开放 PDF 归档提供 HTTP 夹具；`tests/test_literature_skill.py` 覆盖分层去重、引用核验、Markdown、Excel、JSON、归档清单和验证导出。全部夹具在离线测试中运行。
