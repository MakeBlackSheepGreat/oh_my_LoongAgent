# 现有 Task 问题审计报告

**审计日期**：2026-08-01  
**审计范围**：task0-task8 全部代码、spec 与测试  
**审计方法**：逐文件对照 spec 验收条件与代码实际状态，检查依赖管线是否接通

---

## P0 — 必须修复（管线中断）

### 1. 草案审批后运行时从未执行

**所在 task**：task5.5 / task5.10  
**位置**：
- [`internal/workbench/task_drafts.go`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/internal/workbench/task_drafts.go#L200-L233) `ApproveDraft` 调用 `HarnessStore.CreateRun` 后返回
- [`internal/workbench/handlers_task_drafts.go`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/internal/workbench/handlers_task_drafts.go#L107-L121) `handleApproveDraft` 将 `run_id` 返回给前端

**问题**：`ApproveDraft` 创建了 `RunState`（状态 `created`），但从未调用 `HarnessRuntime.Run()`。任务被批准的瞬间 run 就永远停在 `created` 状态，B0-B4 策略、验证器、工具执行器全部闲置。

**spec 验收条件 5** 写的是"调用 HarnessStore.CreateRun 创建 RunState，响应返回 run_id"，这在代码层面确实满足了，但实际业务闭环要求"创建后执行"——否则用户批准的任务永远得不到执行。

**影响**：整个 Agent 执行管线形同虚设。没有真实任务能被运行，D-011 审批工作流只有前半段。

### 2. 运行时与策略未被接入 cmd/server

**所在 task**：task6.3 / task6.4 / task6.5  
**位置**：
- [`cmd/server/main.go`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/cmd/server/main.go) 没有创建 `HarnessRuntime` 实例
- 没有装配 Provider、Tools、Validators、UsageSink 等 `RuntimeOptions`
- `Server` 结构体（[`server.go`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/internal/workbench/server.go#L12-L20)）不持有 `HarnessRuntime`

**问题**：`HarnessRuntime` 和 B0-B4 策略在 `internal/harness/` 中已实现并通过单元测试（42 个新增测试），但 `cmd/server` 完全不知道它们的存在。`ApproveDraft` 即使想调用 `Run()` 也找不到运行时实例。

**spec 验收条件 4** 说"HarnessRuntime 执行循环完整"，这只在单元测试中成立，在生产代码中从未被触发。

### 3. 验证器注册表从未被初始化连接

**所在 task**：task6.0 / task7.6  
**位置**：
- `cmd/server` 中无 `NewValidatorRegistry` 调用
- `evidence_integrity` 验证器（[`internal/skills/literature_search/evidence.go`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/internal/skills/literature_search/evidence.go#L13)）从未被注册到全局验证器注册表
- `ServerOptions` 没有 `Validators` 字段

**问题**：`evidence_integrity` 验证器虽然实现了 `RegisterEvidenceValidator(reg)` 方法，但没有任何地方调用它。`cmd/server` 没有初始化 `ValidatorRegistry`，也没有将其传递给 `HarnessRuntime`。

**spec 验收条件 2** 说"内建验证器覆盖工件存在性、Schema、引用完整性、预算约束"，**验收条件 5** 说"引用核验只允许指向候选池内已归档证据"，但这些验证器在运行时中从未被启用。

---

## P1 — 重要缺陷

### 4. literature_search 的 ACL 来源无公开搜索 API

**所在 task**：task7.3  
**位置**：[`internal/skills/literature_search/manifest.go`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/internal/skills/literature_search/manifest.go#L37)

```go
"acl": "", // ACL Anthology 暂无公开搜索 API，留空表示跳过
```

**问题**：spec 描述"四类来源（arXiv/OpenAlex/Crossref/ACL）"，但 ACL 的 base URL 为空字符串。`aclSource.Search` 在没有 base URL 时会返回空结果或错误。虽然 spec 中已注明"ACL 无公开搜索 API，客户端以通用 JSON 协议预留"，但：
- 前端 `/api/skills` 返回的 `description` 写的是"跨 arXiv/OpenAlex/Crossref/ACL 搜索文献"
- 用户体验上四类来源实际上只有三类可用
- 应当在 Manifest 中显式标记 ACL 为"不可用"或提供替代方案

### 5. 前端 SSE 重连未处理 401 场景

**所在 task**：task8.6  
**位置**：[`web/src/stores/workbench.ts`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/web/src/stores/workbench.ts#L365-L377) `onerror` 回调

**问题**：当会话过期（401）时，SSE 连接 `/api/events` 被 AuthMiddleware 拒绝，`onerror` 触发 3 秒自动重连。但此时 EventSource 收到的 HTTP 状态码是 401，前端没有检测到这一点，会无限重连 401 端点。应当：
- 在 `onerror` 中检测是否 401（通过检查 EventSource 实例的 `status` 属性）
- 如果是 401，停止重连并通知用户"会话已过期，请重新登录"

### 6. 前端发送消息无 loading 状态

**所在 task**：task8.5  
**位置**：[`web/src/stores/workbench.ts`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/web/src/stores/workbench.ts#L184-L195) `sendMessage`

**问题**：`loading` 对象没有 `sendMessage` 字段，发送消息时缺乏 loading 指示。如果后端响应慢，用户无法感知发送状态，可能重复点击造成重复消息。

---

## P2 — 潜在问题

### 7. task9（交叉编译与发布验证）仍为 pending

**所在 task**：task9  
**SPEC.md** 中 task9 状态为 `pending`，但 task5 验证时已通过 4 组合交叉编译（windows/amd64、linux/amd64、linux/arm64、darwin/arm64）。task9 的验收条件已部分满足，至少应将 task9 标记为 `in_progress` 并明确剩余工作。

### 8. 供应商健康检查无自动定时任务

**所在 task**：task5.6  
**问题**：健康检查只在用户手动点击时触发（`GET /api/providers/{id}/health`），没有后台自动检测。如果 provider 在两次手动检查之间宕机，用户无法及时感知。

### 9. 注册失败文案未 i18n 化

**所在 task**：task8.3  
**位置**：[`web/src/views/LoginView.vue`](file:///c:/Users/876762330/Desktop/projects/面向8B参数大模型设计的日常多Agent系统/web/src/views/LoginView.vue) 注册失败时 `registerError` 显示后端原始错误消息

**问题**：后端返回的英文错误消息（如 `"display_name must be <= 64 characters"`）被直接展示给用户，未通过 `t()` 翻译成中文。

---

## 严重程度分布

| 级别 | 数量 | 说明 |
|------|------|------|
| P0 | 3 | 管线中断，任务无法实际执行 — **全部已修复** |
| P1 | 3 | 功能缺陷或用户体验问题 — **全部已修复** |
| P2 | 3 | 优化或未完成工作 — **1 已修复（注册文案 i18n），2 改由 task9 处理** |

## 修复状态

| # | 级别 | 问题 | 修复位置 | 状态 |
|---|------|------|----------|------|
| 1 | P0 | 草案审批后运行时从未执行 | `internal/workbench/task_drafts.go` L235-237 | ✅ 已修复 |
| 2 | P0 | 运行时与策略未被接入 cmd/server | `cmd/server/main.go` L85-142 | ✅ 已修复 |
| 3 | P0 | 验证器注册表从未被初始化连接 | `cmd/server/main.go` L111-115 | ✅ 已修复 |
| 4 | P1 | literature_search ACL 来源无公开 API | `internal/skills/literature_search/manifest.go` L53 | ✅ 已修复 |
| 5 | P1 | 前端 SSE 重连未处理 401 | `web/src/stores/workbench.ts` L393-400 | ✅ 已修复 |
| 6 | P1 | 前端发送消息无 loading 状态 | `web/src/stores/workbench.ts` L30/L72/L192 | ✅ 已修复 |
| 7 | P2 | task9 状态仍为 pending | 待 SPEC.md 更新 | ⏳ 改由 task9 处理 |
| 8 | P2 | 供应商健康检查无自动定时任务 | 待实现 | ⏳ 改由 task9 处理 |
| 9 | P2 | 注册失败文案未 i18n 化 | `web/src/views/LoginView.vue` | ✅ 已修复 |

## 根因分析

所有 P0 问题共享同一个根因：**task5 和 task6 的实现是独立完成的，但 task5 完成后没有将 task6 的运行时接入到应用层**。`ApproveDraft` 只完成了 D-011 的前半段（创建 RunState），后半段（启动执行循环）被遗漏了。`cmd/server/main.go` 在 task5 完成后没有更新来接入 task6 的组件。

## 修复建议

1. 在 `cmd/server/main.go` 中创建 `HarnessRuntime` 实例，装配 Provider、Tools、Validators、UsageSink
2. 在 `internal/workbench/server.go` 中增加 `Runtime` 字段
3. 修改 `ApproveDraft` 或创建新的 `StartRun` 流程：调用 `CreateRun` → 存入数据库 → 异步启动 `HarnessRuntime.Run()`
4. 初始化 `ValidatorRegistry` 并注册 `evidence_integrity` 等验证器
5. 修复 SSE 401 重连逻辑
6. 更新 literature_search 的 Manifest 描述以准确反映 ACL 可用性