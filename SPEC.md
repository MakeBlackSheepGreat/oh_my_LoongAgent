# 项目计划规范与主任务表

**计划版本**：v1.8  
**更新时间**：2026-08-01  
**当前阶段**：P0 Go + Vue 主线从零构建 — task9 多平台发布进行中

## 计划规则

每个大型任务使用稳定编号 `taskN`，并在 `specs/taskN-名称.md` 中拆成可独立验收的小任务 `taskN.0`、`taskN.1`、`taskN.2`。

任务状态只使用：

- `pending`：尚未开始，依赖可能未满足。
- `in_progress`：当前正在推进；同一执行者一次只保留一个。
- `blocked`：存在明确外部阻塞，记录解除条件。
- `completed`：验收条件满足并附有验证证据。
- `cancelled`：目标失效，保留取消原因。

每个任务必须写明目标、输入、输出、依赖、验收条件、状态和验证证据。计划允许实时更新；更新时递增版本或修改更新时间，并在"计划变更记录"写明原因。已完成任务的历史事实不得改写。

> **v1.0 重启说明**：旧 Python/React 主线的 task0-task14 已全部归档（见下方"旧主线归档"）。自 v1.0 起，任务编号从 task0 重新开始，对应 Go 后端 + Vue 3 前端的唯一主线。旧 spec 文件保留在 `specs/archive/` 仅供历史追溯。

## 主任务表

| ID | 大型任务 | 状态 | 详细 Spec | 退出条件 |
| --- | --- | --- | --- | --- |
| task0 | 项目指导文件与技术栈冻结 | completed | `specs/task0-guidance-stack-freeze.md` | RULES v0.3、D-013、D-014、C-007 就绪；Go+Vue+SQLite+多平台多架构边界冻结 |
| task1 | Go 核心契约与存储 | completed | `specs/task1-go-core-contracts.md` | Go struct 覆盖全部契约；SQLite schema 就绪；状态机、版本守卫、内容寻址工件、事件、依赖图与回放可用；CGO_ENABLED=0 编译通过 |
| task2 | Provider 层与 DeepSeek 定制 | completed | `specs/task2-providers-deepseek.md` | OpenAI 兼容 provider 调用可审计；DeepSeek preset 具备 prefix cache 稳定会话与 stale 输出剪枝；健康检查与密钥环境变量读取正确 |
| task3 | 多用户账户与数据隔离 | completed | `specs/task3-accounts-isolation.md` | accounts 表、会话中间件、account_id 过滤、scope 区分与 404 不泄露边界均有验证 |
| task4 | token 计量与公共池视图 | completed | `specs/task4-usage-metering.md` | usage_events 落账、按账户/供应商/时间聚合、前端用量面板与成本显示均通过测试 |
| task5 | workbench 应用层 | completed | `specs/task5-workbench.md` | HTTP 服务入口、AuthMiddleware 路由、项目/会话/消息/任务草案/供应商档案 CRUD、草案 approve→CreateRun、SSE 事件流、embed.FS 静态嵌入与 SPA fallback 全部可用；4 组合交叉编译通过 |
| task6 | 验证器与运行时基线 | completed | `specs/task6-validators-baselines.md` | B0-B4 策略、验证器注册聚合、定向恢复与预算停止在 Go 版本通过测试；errs 子包解除 harness↔providers 导入循环；42 个新增测试全部通过 |
| task7 | Skill adapter 与首发 Skill | completed | `specs/task7-skill-adapter.md` | literature_search Skill 通过 Go adapter 完成搜索、归档、证据与导出；file_organizer 回归样例通过；/api/skills 返回真实注册列表；31 个新增测试全部通过 |
| task8 | Vue 前端工作台与 i18n | completed | `specs/task8-vue-workbench-i18n.md` | Vue 3 前端对接 Go 后端；中英双语可切换并持久化；账户切换、用量面板与语言切换入口完成 |
| task9 | 多平台多架构与发布验证 | in_progress | `specs/task9-cross-platform-release.md` | CGO_ENABLED=0 单静态二进制在 Windows/Linux/macOS × amd64/arm64/loong64/riscv64 全组合可交叉编译；B0-B4 基线通过；发布前 CI 覆盖至少 4 组合；健康检查定时任务实现 |
| task10 | 运行时管线接通与修复 | completed | — | 3 个 P0 全部修复：ApproveDraft→HarnessRuntime.Run() 管线接通、cmd/server 装配运行时组件、验证器注册表初始化与挂载；P1 全部修复：SSE 401 重连、发送消息 loading、ACL 描述修正；P2 注册文案 i18n 已修复；剩余 P2（task9 状态更新、健康检查定时任务）改由 task9 处理 |

## 拆分标准

小任务应在一次连续工作周期内完成，产出一个可检查结果。出现以下情况时继续拆分：

- 同时涉及两个以上独立模块；
- 验收需要多个互不依赖的证据；
- 预计修改范围跨越实现、数据与评测三个领域；
- 任一部分可以独立失败或独立回滚。

## 实时更新流程

1. 开始前将目标小任务改为 `in_progress`，核对依赖。
2. 发现新工作时新增子任务，避免暗中扩大原任务范围。
3. 设计方向变化时更新目标、依赖和验收条件，并记录变更原因。
4. 完成后附上文件、命令、测试或实验记录，再标记 `completed`。
5. 大任务所有必需子任务完成后，更新主任务表并创建下一任务 Spec。

## 旧主线归档（Python/React，已退役）

以下任务属于旧 Python/React 主线，已于 D-014 全部退役。spec 文件移至 `specs/archive/`，仅供历史追溯，不再进入构建、测试或运行路径。

| 旧 ID | 名称 | 退役原因 |
| --- | --- | --- |
| task0 | 规范项目指导文件 | 被 v1.0 task0 替代 |
| task1 | 重新冻结通用 Harness | 边界已继承到 v1.0 |
| task2 | 通用契约 | Python 实现已删除，Go 在 v1.0 task1 重新实现 |
| task3 | 代码迁移 | Python 实现已删除，Go 在 v1.0 task7 重新实现 |
| task4 | 状态与工件 | Python 实现已删除，Go 在 v1.0 task1 重新实现 |
| task5 | 模型适配 | Python 实现已删除，Go 在 v1.0 task2 重新实现 |
| task6 | 工具治理 | Python 实现已删除，Go 在 v1.0 task6 重新实现 |
| task7 | 验证器 | Python 实现已删除，Go 在 v1.0 task6 重新实现 |
| task8 | 运行时与基线 | Python 实现已删除，Go 在 v1.0 task6 重新实现 |
| task9 | B4 与经验 | Python 实现已删除，Go 在 v1.0 task6 重新实现 |
| task10 | 论文搜索 Skill | Python 实现已删除，Go 在 v1.0 task7 重新实现 |
| task11 | 项目中心工作台 | Python/React 实现已删除，Go+Vue 在 v1.0 task5+task8 重新实现 |
| task12 | 评测与发布 | 合并到 v1.0 task9 |
| task13 | 工作台体验与供应商档案 | Python/React 实现已删除，Go+Vue 在 v1.0 task2+task5 重新实现 |
| task14 | Go 后端重写与多用户体系 | 拆分为 v1.0 task1-task9 |

## 计划变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-07-30 | 在 `PROJECT_GOALS.md` 建立 P0-P6 初始路线 |
| v0.2 | 2026-07-30 | 建立 `taskN` 动态计划体系，新增根级 Spec 与分任务目录 |
| v0.3 | 2026-07-30 | 完成 task0.7，纳入来源登记、开源许可与发布检查要求 |
| v0.4 | 2026-07-31 | 将系统明确为通用 Agent Harness；论文搜索调整为首个 Skill Pack，BVAR 固定为 B4 路由策略，并扩展为 task1-task12 的 72 项实施计划。 |
| v0.5 | 2026-07-31 | 完成通用契约、迁移、状态与工件、Provider、工具治理、验证器、B4 与经验模块；当前集中推进 B0-B3 的公平基线。 |
| v0.6 | 2026-07-31 | 将 task11 重构为项目中心 Agent 工作台：多本地项目、聊天草案审批、Skill 插件配置与三栏观察面板成为首发交互边界。task8 完成后启动实现。 |
| v0.7 | 2026-07-31 | 用户要求提前交付 task11。项目中心工作台、受信任根目录、会话草案、Skill 配置、三栏界面、动态端口和验证闭环已完成；task8 的基线验收继续独立推进。 |
| v0.8 | 2026-07-31 | 用户要求工作台交互参考 Codex 与 Claude Code 的项目-任务-对话节奏，视觉调整为轻盈清新的本地工具表面；新增 task13，供应商档案借鉴 CC Switch 的配置档案、激活、环境变量密钥引用与连通性诊断机制。为保持单一当前任务，task8 暂回到 pending。 |
| v0.9 | 2026-08-01 | 用户决定后端以 Go 语言重写并纳入多用户体系，参考 DeepSeek-Reasonix（MIT）的 Go harness 架构；新增 task14，合并推进多用户隔离、token 计量、DeepSeek 定制与 i18n。Python 版本保留为迁移参考与对照基线。见 D-013、C-007。 |
| v0.10 | 2026-08-01 | 用户决定删除全部 Python 代码不做存档，Go 为唯一后端主线；前端从 React 切换到 Vue 3 + TypeScript；RULES.md v0.3 写入多平台（Windows/Linux/macOS）多架构（amd64/arm64/loong64/riscv64）支持与 CGO_ENABLED=0 单静态二进制要求。Python 代码与 React 前端已删除，Vue 骨架已建。见 D-014。 |
| v1.0 | 2026-08-01 | 任务表从零重启。旧 Python/React 主线 task0-task14 全部归档到 `specs/archive/`。新 task0-task9 对应 Go+Vue 唯一主线：task0 技术栈冻结（completed）、task1 Go 核心契约与存储（in_progress，继承原 task14.0+14.1 成果）、task2-task9 依次推进 Provider、多用户、计量、workbench、验证器、Skill、Vue 前端与多平台发布。 |
| v1.1 | 2026-08-01 | task3（多用户账户与数据隔离）完成：Account 契约、accounts 表、会话管理、AuthMiddleware、AccountScoped 数据隔离、provider_profiles scope CRUD 全部就绪；32 个测试通过；CGO_ENABLED=0 build + vet 通过。自实现 ULID（crypto/rand + Crockford Base32）替代 oklog/ulid 外部依赖。 |
| v1.2 | 2026-08-01 | task4（token 计量与公共池视图）完成：usage_events 表、MeterRecorder 落账、四种时间窗口聚合、公共池视图（system scope 用量归属调用方）、用量 API handler（预留挂载点，待 task5 建 server 后挂载）全部就绪；15 个新测试通过（全项目 90 个）；CGO_ENABLED=0 build + vet 通过。 |
| v1.3 | 2026-08-01 | 建立 task5 spec（`specs/task5-workbench.md`），拆分 11 个子任务 task5.0-task5.10；覆盖 HTTP 服务入口、统一响应层与错误映射、认证 API、领域模型扩展（conversations/messages/task_drafts）、项目与会话 API、任务草案审批工作流（D-011 approve→CreateRun）、供应商档案与健康检查、SSE 事件流、embed.FS 静态嵌入与 SPA fallback、端到端测试、鲁棒性检查；明确不实现 task4 计量与 task6+ 运行时但预留接入点；更新主任务表 task5 退出条件。 |
| v1.4 | 2026-08-01 | 建立 task6 与 task7 spec（`specs/task6-validators-baselines.md`、`specs/task7-skill-adapter.md`）。task6：验证器接口与注册表、内建验证器、受限工具执行器（旧工具治理并入）、HarnessRuntime 执行循环、B0-B4 策略（固定单流程/单 Agent 闭环/串行角色复用/固定分支候选/BVAR）、预算停止、定向恢复，10 个子任务 task6.0-task6.9。task7：Skill 接口与注册表、HTTP 工具基础设施、literature_search 首发 Skill（四来源/去重溯源/全文归档/证据核验/四种导出）、file_organizer 回归样例，11 个子任务 task7.0-task7.10；明确领域边界 D-009、导出用 CSV 兼容 Excel 规避网络依赖。 |
| v1.5 | 2026-08-01 | task5（workbench 应用层 HTTP 服务）完成：cmd/server 入口（flag/env 装配 + 优雅关闭 + embed.FS 静态嵌入 + SPA fallback）、统一响应层与错误映射、认证 API（HttpOnly cookie）、项目/会话/消息/任务草案 CRUD、草案 approve→CreateRun（D-011）、供应商档案与健康检查 API、SSE 事件流（EventHub 按账户过滤）、/api/skills 占位挂接；task4 用量 API 挂载；11 个端到端测试通过（全项目 113 个）；4 组合交叉编译（windows/amd64、linux/amd64、linux/arm64、darwin/arm64）通过；CGO_ENABLED=0 build + vet 通过。 |
| v1.6 | 2026-08-01 | 建立 task8 spec（`specs/task8-vue-workbench-i18n.md`），拆分 12 个子任务 task8.0-task8.11；覆盖前端工程化（vue-router/i18n/pinia）、API client 与鉴权流、中英双语 i18n（持久化到账户 locale）、后端配套补充（GET /api/accounts 公开账户列表 + PATCH /api/auth/me 更新 locale）、三栏工作台布局、项目/会话/聊天、草案审批与 SSE、供应商与用量面板、技能面板、视觉打磨（SOUL.md 运行工作台表面）、构建与端到端验证、鲁棒性检查；明确 i18n 范围限界（仅 UI 文案，不做模型 system prompt 切换）。 |
| v1.7 | 2026-08-01 | 全面审计现有 task0-task8，发现 9 个问题（3 P0、3 P1、3 P2）。P0：草案审批后运行时从未执行、运行时与策略未被接入 cmd/server、验证器注册表从未初始化连接。P1：ACL 来源无公开 API、SSE 重连未处理 401、发送消息无 loading。P2：task9 状态未更新、供应商健康检查无自动定时任务、注册文案未 i18n。新增 task10 修复管线。报告写入 `guidance/ISSUES.md`。 |
| v1.8 | 2026-08-01 | task10 管线修复 7/9 完成：3 个 P0 全部修复（ApproveDraft→Run、cmd/server 运行时装配、验证器注册表挂载），3 个 P1 全部修复（SSE 401 重连、sendMessage loading、ACL 描述），1 个 P2 修复（注册文案 i18n）。task10 标记 completed。剩余 P2（task9 状态更新、健康检查定时任务）改由 task9 处理。task9 推进中。 |
| v1.9 | 2026-08-01 | ① 用户要求自查 dev.ps1：发现脚本从未成功运行（PowerShell C 风格注释运行时崩溃 + Start-Process 同文件重定向报错），修复并完整验证（后端/前端/代理全 200，taskkill /T 整树清理）；README 本地运行段落同步修正；教训登记 `guidance/REFLECTIONS.md` R-001/R-002。② task9 子任务盘点：task9.1 前端构建产物已存在待复验；task9.0 交叉编译、task9.2 健康检查定时任务、task9.3 CI、task9.4 发布清单、task9.5 测试覆盖均未做；dist/ 无产物、无 .github/workflows、无 RELEASE_CHECKLIST.md。下一个推进子任务：task9.0（依赖 task0-task10 已满足）。③ 前端构建复验：Windows 原生与 WSL bash 两环境 `npm run build` 均成功（vue-tsc + vite build，EXIT=0，产物 dist/index.html + assets 一致）；WSL 首次构建失败系 lightningcss 平台绑定缺失（node_modules 缺 linux 绑定），`npm install` 按 lock 补装后修复——task9.1 构建验证两环境均可执行。 |
| v1.10 | 2026-08-01 | 用户指出前端质量差（白屏/组件不显示/供应商无反应/无账号密码），实测定位 4 个 bug 并全部修复：① `InitAll` 漏调 `InitUsageTable`（usage API 500）→ 补上并加迁移顺序修复（旧库先补列再建索引）；② 列表 API 空数据返回 JSON `null`（Go nil slice）导致前端 `null.length` 渲染崩溃 → 5 处 List 方法返回 `[]` + 前端 store 统一 `?? []` 兜底；③ 前端创建供应商不传 `profile_id` 导致 400 → 服务端生成 ULID；④ 无账号密码体系 → 完整实现：accounts 表加 username/password_hash 列（含旧库迁移与 username 回填）、标准库 PBKDF2-HMAC-SHA256 密码哈希（crypto/pbkdf2，210000 迭代）、注册/登录 handler 改造（统一 401 防枚举）、LoginView 重写为账号密码表单、i18n 中英文案、测试全部更新。playwright 全流程实测通过：注册→工作台无崩溃→供应商一键添加 201→usage 200→登出→错误密码提示→正确密码登录成功。教训登记 `guidance/REFLECTIONS.md` R-003。 |
| v1.11 | 2026-08-01 | security_review（认证变更专项）结论：minor concerns，无 critical/high。已修复：① 登录/禁用账户分支统一 dummy PBKDF2 抹平时序（含空 hash 迁移账户）；② 登录/注册端点密码上限 128 字符（rune 计，防未认证端点超长输入 CPU DoS）；③ 迁移 username 回填截断 29 预留后缀空间、短于 3 字符回退 "user"。已接受残余（本地工具场景）：注册 409 用户名枚举（UX 优先）、会话 cookie 无 Secure（本地 http 必需，HTTPS 部署时需配置化）、畸形 hash 早退时序（需 DB 写权限）、unique 错误方言匹配、dataDir 0o755 依赖 umask、无 CSP。 |
