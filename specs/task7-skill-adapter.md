# task7：Skill adapter 与首发 Skill

**状态**：completed  
**目标**：在 Go 版本建立 Skill 注册与运行时适配层，交付首发 `literature_search` Skill（四类来源搜索、去重溯源、全文归档、证据核验、导出）与 `file_organizer` 领域无关性回归样例，验证核心不依赖任何垂直领域。

## 输入

- task1（Go 核心契约）已就绪：`internal/harness/contracts.go` 提供 SkillManifest（SkillID/Version/Title/Description/InputSchema/OutputArtifactKinds/RequiredTools/RequiredValidators/DefaultBudget/Metadata）与 TaskContract。
- task6（验证器与运行时基线）需先完成：受限工具执行器（Policy 检查）、验证器注册表、HarnessRuntime 执行循环；Skill 的工具与验证器挂载到该运行时。
- task5（workbench HTTP 服务）需先完成：`/api/skills` 占位 handler（task5.7 设计边界中预留）在 task7 变为真实注册列表。
- 旧主线领域语义（D-014 已删 Python，从归档 task10-literature-search-skill 追溯）：Skill Manifest 与任务输入、四类来源工具（arXiv/OpenAlex/Crossref/ACL）、去重与来源溯源、开放全文归档、证据与引用核验、四种导出（Markdown/Excel/JSON/归档清单）。
- 架构约束（D-009/D-013/D-014）：论文、PDF、DOI、arXiv、Excel 导出等领域能力必须位于 Skill 层；核心层不依赖领域对象；`literature_search` 是首发 Skill Pack 与评测场景；至少一个模拟文件整理 Skill 回归验证核心领域无关性。
- SOUL.md 代码口味：模块小而聚焦、边界按领域责任划分、适配层不泄漏到核心领域逻辑。

## 输出

- `internal/skills/registry.go`：Skill 接口（Manifest()/Execute(ctx, task, env) → 工件/事件/验证结果）、Registry（Register/Lookup/List）。
- `internal/skills/adapters/`：HTTP 客户端与 httptest 夹具基础设施。
- `internal/skills/literature_search/`：Manifest、来源工具、去重、归档、证据核验、导出。
- `internal/skills/file_organizer/`：领域无关性回归样例（模拟文件整理）。
- `internal/skills/*_test.go`：全部离线测试（httptest 夹具）。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task7.0 | Skill 接口与注册表 | completed | `registry.go` 定义 `Skill` 接口（Manifest() *harness.SkillManifest；Execute(ctx, req *Request, env *Env) (*Result, error)）；`Registry` 支持 Register/Lookup/List；重复注册 CONFLICT；未知 NOT_FOUND；`/api/skills` 返回真实注册列表（task5 占位 handler 挂接） |
| task7.1 | Skill 工具基础设施 | completed | `adapters/` 提供带超时/取消的 HTTP client、httptest 夹具模式（离线测试用）、响应大小上限（防恶意负载）、JSON 解码错误标准化 |
| task7.2 | literature_search Manifest 与输入校验 | completed | `SkillManifest`：skill_id=literature_search、InputSchema（研究问题/来源/预算/导出要求）、OutputArtifactKinds、RequiredTools、RequiredValidators；Execute 前校验任务输入（objective 非空、来源集合合法、预算边界），失败返回 VALIDATION_ERROR |
| task7.3 | 四类来源工具 | completed | arXiv/OpenAlex/Crossref/ACL 四个来源客户端（各自 httptest 夹具）：搜索返回结构化记录（ID/标题/作者/年/DOI/摘要/URL/开放全文链接）；响应超时 10 秒可配置；网络错误映射 PROVIDER_UNAVAILABLE |
| task7.4 | 去重与来源溯源 | completed | DOI、ID、标题相似度分层合并：同一 DOI/ID 直接合并；标题归一化（小写/去标点/压缩空白）相似度 ≥ 阈值合并；合并保留全部来源与证据链；溯源记录（每个候选来自哪些来源） |
| task7.5 | 开放全文归档 | completed | 下载开放全文（限 MIME：pdf/plain/html；限大小，默认 ≤ 50MB）；保存 MIME、大小、SHA-256 哈希、版权状态、来源 URL；归档清单（artifact JSON）可回放 |
| task7.6 | 证据与引用核验 | completed | 引用只能指向候选池内已归档证据（D-003 工件引用）；未核验引用返回验证失败与 Findings；证据核验验证器挂载到 harness Registry |
| task7.7 | 四种导出 | completed | Markdown（结构化的证据包报告）、JSON（机器可读全量）、CSV（Excel 兼容表格）、归档清单（manifest.json 可回放）；导出工件写入 harness 工件库（SHA-256 内容寻址） |
| task7.8 | file_organizer 回归样例 | completed | 模拟文件整理 Skill：扫描目录→按规则分类（模拟移动/复制，不触碰真实文件系统外路径）→输出整理计划工件；用 httptest 与临时目录离线验证；证明核心运行时不依赖领域对象（D-009） |
| task7.9 | 测试覆盖 | completed | registry_test（注册/查找/列表/冲突）、sources_test（四来源夹具）、dedup_test（分层合并）、archive_test（全文归档）、evidence_test（引用核验）、export_test（四种导出）、file_organizer_test（回归样例）；全部离线 |
| task7.10 | 鲁棒性与死代码检查 | completed | 风格与 harness/providers/workbench/skills 一致；无未使用导出符号/不可达分支/未引用包；HTTP 请求全部带 context 与超时；无裸 goroutine 泄漏；CGO_ENABLED=0 build + vet 通过 |

## 验收条件

1. `internal/skills/` 提供 Skill 接口与 Registry；`/api/skills` 返回真实注册列表。
2. literature_search 通过四类来源（arXiv/OpenAlex/Crossref/ACL）完成搜索，全部离线测试用 httptest 夹具。
3. 去重按 DOI/ID/标题相似度分层合并，保留完整来源与证据链。
4. 开放全文归档保存 MIME、大小、SHA-256、版权状态、来源 URL，归档清单可回放。
5. 引用核验只允许指向候选池内已归档证据；未核验引用返回验证失败与 Findings。
6. 四种导出（Markdown/JSON/CSV/归档清单）工件写入 harness 工件库（SHA-256 内容寻址）。
7. file_organizer 模拟 Skill 回归通过，证明核心运行时领域无关（D-009）。
8. `internal/harness/` 与 `internal/skills/` 边界清晰：Skill 不修改核心契约，仅挂载工具与验证器。
9. `CGO_ENABLED=0 go build` 与 `go vet` 通过；全部测试通过。

## 设计边界

- **领域边界（D-009）**：论文、PDF、DOI、arXiv、Excel 导出等领域能力全部位于 `internal/skills/literature_search/`；`internal/harness/` 只提供 SkillManifest 契约、工具执行器与验证器注册表，不包含任何领域对象。
- **导出格式**：四种导出为 Markdown、JSON、CSV（Excel 兼容）、归档清单。因 Go 侧引入 excelize 存在网络依赖风险（本机 GOPROXY 受限，task3 已遇 go get 失败），Excel 以 CSV 形式交付（Excel 可直接打开），避免第三方依赖；若后续网络可用再评估 excelize 升级为原生 .xlsx。
- **来源工具复用**：task7.1 的 HTTP client 基础设施被四类来源共用；各来源只实现"搜索→结构化记录"的最小适配，不共享领域逻辑。
- **去重阈值**：标题相似度用归一化 Jaccard 或等价的确定函数，阈值常量可配置（默认 0.9）；同一 DOI/ID 直接合并（优先级最高）。
- **全文归档限制**：默认单文件 ≤ 50MB、MIME 白名单（application/pdf/text/plain/text/html）；超出或未知 MIME 仅记录元数据不下载内容。
- **验证器挂载**：证据核验验证器以领域 Skill 的 RequiredValidators 声明，注册进 task6 的 harness Registry；核心运行时按 SkillManifest 自动挂载。
- **回归样例**：file_organizer 使用临时目录 + 模拟移动/复制（不实际移动用户文件），证明"核心可执行任何领域 Skill"而不依赖论文/PDF 对象。
- **不强化的范围**：本任务不实现 MCP 插件协议适配（Reasonix 的 plugin 体系参考；当前以 Go 接口注册即可）；B4 经验编译与 Skill Card 模型不在本任务（task6.5 预留扩展点）。

## 子任务依赖

- task7.0 依赖 task1（SkillManifest 契约）+ task5（/api/skills 挂接点）。
- task7.1 依赖 task7.0（Skill 需要工具基础设施）。
- task7.2 依赖 task7.0（Manifest）+ task1（TaskContract）。
- task7.3 依赖 task7.1（HTTP 基础设施）。
- task7.4 依赖 task7.3（来源记录）。
- task7.5 依赖 task7.3（开放全文链接）+ task1（工件库）。
- task7.6 依赖 task7.4（候选池）+ task6.0（验证器 Registry）。
- task7.7 依赖 task7.4-task7.6（数据就绪）+ task1（工件库）。
- task7.8 依赖 task7.0（Skill 接口）+ task6.2（工具执行器）。
- task7.9 依赖 task7.0-task7.8 全部完成。
- task7.10 依赖 task7.9（测试通过后做最终检查）。
- task7.3 四个来源可并行实现；task7.8 与 task7.2-task7.7 可并行。

## 影响范围

- Affected specs: task5（/api/skills 从占位变真实）、task6（验证器注册表被领域验证器挂载）、task8（Vue 前端展示 Skill 列表与运行）、task9（发布验证含 Skill 回归）。
- Affected code: `internal/skills/`（新建）、`internal/harness/validators.go`（被领域验证器挂载，仅注册不修改）、`internal/workbench/handlers_*.go`（/api/skills 挂接）。
- 依赖 task1（SkillManifest/工件库）、task5（HTTP 挂接点）、task6（工具执行器/验证器注册表）。
- 不影响 `internal/harness/` 与 `internal/providers/` 现有代码（只读引用或注册挂载）。

## 验证证据

- 新增 `internal/skills/`：`registry.go`（Skill 接口/Request/Env/Result/Registry）、`adapters/client.go`（超时/取消/大小上限/错误标准化，含 DoWithLimit 供全文归档用）。
- 新增 `internal/skills/literature_search/`：`manifest.go`（Manifest + 输入校验 + 主流程编排）、`sources.go`（arXiv Atom/OpenAlex/Crossref/ACL 四来源）、`dedup.go`（DOI/ID/标题相似三层去重 + 溯源）、`archive.go`（≤50MB MIME 白名单全文归档 + SHA-256）、`evidence.go`（evidence_integrity 验证器）、`export.go`（Markdown/JSON/CSV/归档清单）、`util.go`。
- 新增 `internal/skills/file_organizer/organizer.go`：只读扫描分类产出整理计划，证明核心领域无关（D-009）。
- workbench：`ServerOptions.Skills` 注入 + `handleListSkills` 返回真实注册列表（nil 安全）。
- 新增测试 31 个：registry_test（4）、sources_test（6）、dedup_test（5）、manifest_test（5，含 Execute 端到端）、archive_test（3）、export_test（2）、file_organizer_test（4）、workbench server_test（2）。
- 验证命令：`go build ./...`、`go vet ./...`、`go test ./...`、`CGO_ENABLED=0 go build ./...` 全部通过。
- 设计实现说明：ACL 无公开搜索 API，客户端以通用 JSON 协议预留（测试由 httptest 夹具驱动）；Excel 导出以 UTF-8 CSV 交付规避 excelize 网络依赖（spec 设计边界）；去重第二层仅合并同来源重复 ID，单条记录留给标题相似层，避免误拆跨来源同标题记录。

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task7 spec；拆分 11 个子任务 task7.0-task7.10；明确领域边界（D-009）、导出格式用 CSV 兼容 Excel 规避网络依赖、去重分层策略、全文归档限制、file_organizer 回归样例；不强化的范围（MCP 插件协议、B4 经验编译） |
| v0.2 | 2026-08-01 | task7.0-task7.10 全部完成并标记 completed；补齐验证证据与设计实现说明（ACL 协议预留、归档工件加入 Result、去重第二层单条不拆组） |
