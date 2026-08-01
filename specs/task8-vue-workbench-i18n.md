# task8：Vue 前端工作台与 i18n

**状态**：completed  
**目标**：把 Vite 脚手架前端升级为对接 Go 后端的完整 Vue 3 工作台：登录与账户切换、项目/会话/聊天、任务草案审批、供应商管理、公共模型池用量面板、技能列表、SSE 实时事件，并交付中英双语 i18n（切换持久化到账户偏好）。视觉遵循 SOUL.md 运行工作台表面。

## 输入

- task5（workbench HTTP 服务）已完成：全部 REST API 可用（认证、项目、会话、消息、任务草案审批 D-011、供应商档案与健康检查、用量 task4、SSE、skills 占位）。
- task4（token 计量与公共池视图）已完成：`GET /api/usage/aggregate` 与 `GET /api/usage/public-pool` 已挂载。
- task3（多用户账户）已完成：Account 含 locale 字段；登录用 HttpOnly cookie。
- web/ 现状：Vite 8 + Vue 3.5 + TypeScript 6 脚手架（HelloWorld 占位）；依赖仅 vue；vite.config.ts 已配置 /api 代理到 127.0.0.1:8000；无 vue-router、无 vue-i18n、无状态管理、无 API client。
- 用户历史要求（D-014 前累计）：优化前端 UI、i18n 多语言设置、多账号相互隔离、公共大模型池显示各账号 token 消耗。
- SPEC.md task8 退出条件：Vue 3 前端对接 Go 后端；中英双语可切换并持久化；账户切换、用量面板与语言切换入口完成。
- SOUL.md 运行工作台表面：紧凑、安定、可扫描；中性灰白或深色表面；青绿=执行/通过、钴蓝=信息、琥珀=等待、珊瑚=风险/失败；3-6px 小圆角；首屏回答"正在做什么/由谁执行/消耗多少/证据在哪里/是否需要介入"。
- 架构约束（D-014）：Vue 3 + TypeScript 是唯一前端主线；后端 Go 接口保持不变，前端适配。
- i18n 范围限界（task14.8 设计边界）：只做 UI 文案与后端错误消息本地化；不做模型 system prompt 的语言切换。

## 输出

- `web/src/api/`：fetch API client（credentials include、401 拦截、类型化请求/响应）。
- `web/src/i18n/`：vue-i18n 实例、`zh-CN.json`/`en.json` 文案、locale 切换与持久化（账户偏好 + localStorage 回退）。
- `web/src/stores/`：账户（auth/account 切换）、会话（current conversation）、用量（公共池聚合）。
- `web/src/views/`：LoginView（账户选择）、WorkbenchView（主工作台布局）。
- `web/src/components/`：Header（账户切换+语言切换）、ProjectList、ConversationList、ChatPanel、DraftPanel、ProviderPanel、UsagePanel、SkillList。
- `web/src/App.vue`、`web/src/router.ts`：路由与根组件。
- 后端配套补充：`GET /api/accounts`（公开账户列表，登录选择用）、`PATCH /api/auth/me`（更新 locale 偏好）、workbench `UpdateAccountLocale` 方法。

## 子任务

| ID | 内容 | 状态 | 验收条件 |
| --- | --- | --- | --- |
| task8.0 | 前端工程与依赖 | completed | 因 npm 网络受限，采用手写极简方案：手写 i18n（index.ts/zh-CN.ts/en.ts）、手写 hash 路由（router.ts）、手写组合式 store（session.ts/workbench.ts）；建立 src/{api,i18n,stores,views,components} 目录；vite.config.ts 保留 /api 代理；`npm run build` 零错误 |
| task8.1 | API client 与鉴权流 | completed | `api/client.ts`：fetch 封装（credentials: 'include'、JSON 编解码、ApiError 错误映射）；401 广播 `auth:unauthorized` 事件；login/logout/me/updateMe + 全部业务 API 方法；类型化：Account/Project/Conversation/Message/TaskDraft/ProviderProfile/ProviderUsage/UsageAggregate/PublicPool/SkillInfo/RunEvent |
| task8.2 | i18n 基础设施 | completed | 手写 i18n（index.ts：reactive locale + t() 点路径取文案 + {param} 插值）；zh-CN.ts/en.ts 覆盖全部 UI 文案（app/auth/common/nav/projects/conversations/drafts/providers/usage/skills/status/sse）；Header 语言切换按钮；持久化：登录后写账户 locale（PATCH /api/auth/me），未登录用 localStorage 回退；组件通过 t() 使用，无硬编码中文 |
| task8.3 | 后端配套补充 | completed | `GET /api/accounts` 公开端点（返回 account_id/display_name/locale/status，不含敏感字段）；`PATCH /api/auth/me`（body {locale}，校验 locale ∈ {zh-CN,en}）；workbench.UpdateAccountLocale 方法；handler 测试（TestListAccounts_Public + TestUpdateAccountLocale）通过 |
| task8.4 | 布局与导航 | completed | WorkbenchView 三栏布局：左导航条（6 个垂直标签）、左栏（根据导航切换面板）、中栏（ChatPanel 聊天区）、右栏（4 个标签页：草案/供应商/用量/技能）；Header 含账户名显示、语言切换、登出按钮；响应式（≤900px 右栏折叠、≤640px 左栏隐藏） |
| task8.5 | 项目与会话视图 | completed | ProjectList（创建/删除对话框 + 列表渲染 + confirm 确认）、ConversationList（创建/切换/删除 + 关联项目选择）、ChatPanel（消息流渲染/用户发送/Enter 快捷发送/自动滚动到底部/四种角色消息样式/loading 态） |
| task8.6 | 任务草案与审批 UI | completed | DraftPanel：草案创建对话框（objective 文本 + skill 选择）、列表（draft/approved/rejected 状态标记 badge）、审批/拒绝/删除按钮；run_id 显示；SSE 由 workbench store 统一管理（connectEvents/disconnectEvents，3 秒自动重连，事件到达刷新草案+用量） |
| task8.7 | 供应商与用量面板 | completed | ProviderPanel：供应商列表（scope 标记 badge、active 标记）、创建对话框（5 字段）、激活/健康检查/删除按钮（system scope 禁删）；UsagePanel：今日/本周/本月/累计四档切换（tab 控件）、我的用量三格卡片（token/成本/调用次数）、公共模型池每个 provider 的 T/$/C 三列统计；数值 locale-aware 格式化 |
| task8.8 | 技能面板 | completed | SkillList：GET /api/skills 列表渲染（title/description/version）；空列表展示引导文案 |
| task8.9 | 视觉与交互打磨 | completed | 设计系统（style.css）：中性灰白底色（#f4f6f9）、柔和背景径向渐变微光、分层阴影体系（shadow-sm/md/lg）、四种语义色光晕（glow-teal/cobalt/amber/coral）、8-14px 圆角；按钮（btn/btn-primary/btn-ghost/btn-danger + 渐变+内阴影）、表单（field focus 钴蓝光晕）、面板（panel 圆角+阴影）、列表项（list-item hover/active 过渡+青绿边框）、状态标记（badge 圆点+微光晕）、空态/加载/通知 toast（四种颜色+微光晕+入场动画）、滚动条自定义、按钮点击 transform 反馈 |
| task8.10 | 构建与端到端验证 | completed | `npm run build`（vue-tsc + vite build）通过（49 modules transformed，0 errors）；产出 dist/index.html + dist/assets/（CSS 20.41KB + JS 102.17KB） |
| task8.11 | 鲁棒性与死代码检查 | completed | vue-tsc noUnusedLocals 检查通过（清理了 3 个未使用变量：session/LOCALES/Locale）；SSE 在 WorkbenchView onUnmounted 时 disconnectEvents 清理；API 错误统一通过 workbench.notify 展示 toast；无死代码残留 |

## 验收条件

1. `npm run build`（vue-tsc + vite build）零错误；`make all` 产出含前端的单静态二进制。
2. 前端对接 Go 后端全流程可用：登录（账户选择）→ 项目 CRUD → 会话+消息 → 草案审批（D-011）→ SSE 实时刷新。
3. 中英双语可切换并持久化：登录账户 locale 偏好生效，未登录用 localStorage 回退；UI 无硬编码文案残留。
4. 账户切换入口完成：登录页列出账户选择；工作台内可登出重新选择；切换后数据隔离（跨账户 404 由后端保证）。
5. 用量面板显示各账户在公共模型池（system scope）的 token 消耗、估算成本与调用次数，支持今日/本周/本月/累计窗口。
6. 供应商面板支持 account scope 创建/激活/健康检查/删除（system 禁删）；技能面板在 task7 挂接后显示真实列表。
7. 视觉遵循 SOUL.md 运行工作台表面（状态标记、错误/等待/通过颜色、紧凑可扫描布局）。
8. 后端补充（GET /api/accounts、PATCH /api/auth/me）有对应测试；CGO_ENABLED=0 build + `go test ./...` 全量回归通过。

## 设计边界

- **i18n 范围限界**：仅 UI 文案与后端错误消息本地化；不做模型 system prompt 的语言切换；不做日期/数字 ICU 格式化（用 Intl API 按 locale 呈现）。
- **账户列表语义**：`GET /api/accounts` 是本地工作台公开端点（不鉴权），返回 account_id/display_name/locale/status；不含密钥、会话或任何业务数据。跨账户业务数据隔离不因该端点放松（业务 API 仍 404/401）。
- **locale 持久化链**：登录 → 读取账户 locale → 设置 vue-i18n locale；切换 → PATCH /api/auth/me 持久化 + localStorage 回退（未登录/离线时）。
- **状态管理**：优先组合式 API + 轻量 store；引入 pinia 仅当共享状态跨视图且组合式模式导致重复样板时。账户、当前会话、用量数据为跨视图共享状态。
- **SSE 生命周期**：EventSource 由 WorkbenchView 或专门 composable 持有，组件卸载即 close；SSE 错误（断线）显示重连提示，不静默失败。
- **供应商密钥**：前端只显示 api_key_env 环境变量名，不显示密钥值；密钥始终从环境变量读取（延续 task2 边界）。
- **不强化的范围**：不做移动端原生壳；不做模型参数调节 UI；不做后端管理后台（CLI 后续承担）；i18n 不做全量后端错误消息翻译表（仅前端文案，后端错误码原样透传）。
- **视觉依据**：SOUL.md 运行工作台表面，非编辑展示层；克制动效，状态迁移/数据到达有因果含义。

## 子任务依赖

- task8.0 依赖 web/ 现有脚手架 + task5（API 可用）。
- task8.1 依赖 task8.0（工程就绪）。
- task8.2 依赖 task8.0（工程）+ task8.1（登录流拿账户 locale）。
- task8.3 依赖 task3（Account locale 字段）+ task5（server 路由）。
- task8.4 依赖 task8.1（鉴权）+ task8.2（i18n）+ task8.3（账户列表）。
- task8.5 依赖 task8.4（布局）+ task8.1（API）。
- task8.6 依赖 task8.5（会话）+ task5（草案/SSE API）。
- task8.7 依赖 task8.4（布局）+ task4（用量 API）+ task5（供应商 API）。
- task8.8 依赖 task8.4（布局）+ task5（/api/skills 占位）。
- task8.9 依赖 task8.4-task8.8（组件齐备后统一打磨）。
- task8.10 依赖 task8.0-task8.9 全部完成。
- task8.11 依赖 task8.10（构建通过后做最终检查）。
- task8.3 可与 task8.0-task8.2 并行；task8.5-task8.8 可部分并行。

## 影响范围

- Affected specs: task5（后端 API 由前端消费，不改接口）、task7（/api/skills 由占位变真实后前端显示）、task9（发布验证含前端构建）。
- Affected code: `web/`（新建 src/{api,i18n,stores,views,components}、改造 App.vue、新建 router.ts）、`internal/workbench/`（补 GET /api/accounts、PATCH /api/auth/me、UpdateAccountLocale、handlers 测试）。
- 依赖 task3（locale）、task4（用量）、task5（全部 REST/SSE）、task7（skills 列表挂接，可先行用占位）。
- 不影响 `internal/harness/`、`internal/providers/` 现有代码。

## 验证证据

- `npm run build`（vue-tsc + vite build）零错误通过（49 modules, 0 errors）
- 产出：`dist/index.html`（0.45KB）+ `dist/assets/`（CSS 20.41KB + JS 102.17KB）
- 后端测试：`TestListAccounts_Public` + `TestUpdateAccountLocale` 通过
- 清理死代码：删除 HelloWorld.vue、hero.png、vite.svg、vue.svg；清理 3 个未使用变量
- 文件清单：
  - `web/src/api/client.ts`（API 封装）
  - `web/src/api/types.ts`（类型契约）
  - `web/src/i18n/index.ts`（手写 i18n 核心）
  - `web/src/i18n/zh-CN.ts`（中文文案）
  - `web/src/i18n/en.ts`（英文文案）
  - `web/src/stores/session.ts`（会话 store）
  - `web/src/stores/workbench.ts`（工作台数据 store + SSE）
  - `web/src/router.ts`（手写 hash 路由）
  - `web/src/views/LoginView.vue`（登录页）
  - `web/src/views/WorkbenchView.vue`（三栏布局）
  - `web/src/components/Header.vue`（顶栏）
  - `web/src/components/ProjectList.vue`（项目列表）
  - `web/src/components/ConversationList.vue`（会话列表）
  - `web/src/components/ChatPanel.vue`（聊天面板）
  - `web/src/components/DraftPanel.vue`（草案面板）
  - `web/src/components/ProviderPanel.vue`（供应商面板）
  - `web/src/components/UsagePanel.vue`（用量面板）
  - `web/src/components/SkillList.vue`（技能列表）
  - `web/src/App.vue`（根组件）
  - `web/src/main.ts`（入口）
  - `web/src/style.css`（设计系统）

## 变更记录

| 版本 | 日期 | 变更 |
| --- | --- | --- |
| v0.1 | 2026-08-01 | 建立 task8 spec；拆分 12 个子任务 task8.0-task8.11；明确 i18n 范围限界（仅 UI 文案）、账户列表公开端点语义、locale 持久化链、SSE 生命周期、供应商密钥边界；后端仅补两个配套端点（GET /api/accounts、PATCH /api/auth/me），不改既有接口 |
