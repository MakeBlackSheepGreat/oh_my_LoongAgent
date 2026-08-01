<div align="center">

# ⚡ oh-my-loongAgent

### 面向 ≤12B 参数模型的轻量多 Agent 编排引擎

**以类型化契约约束模型输出 · 验证器驱动预算分配 · 全程可回放可审计**

</div>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3-42b883?logo=vuedotjs&logoColor=white" alt="Vue 3">
  <img src="https://img.shields.io/badge/TypeScript-5-3178c6?logo=typescript&logoColor=white" alt="TypeScript">
  <img src="https://img.shields.io/badge/SQLite-pure&nbsp;Go-003b57?logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/CGO-0%20static-d22" alt="CGO_ENABLED=0">
  <img src="https://img.shields.io/badge/platforms-8%20combos-5c6bc0" alt="8 platforms">
  <img src="https://img.shields.io/badge/license-Apache%202.0-blue" alt="Apache 2.0">
</p>

---

```
                    ┌─────────────────────────────────────────────────┐
                    │                   oh-my-loongAgent               │
                    │        小模型 · 大能力 · 零依赖 · 全可审计        │
                    └─────────────────────────────────────────────────┘
                                        │
          ┌─────────────────────────────┼──────────────────────────────┐
          ▼                             ▼                              ▼
   ┌──────────────┐           ┌────────────────┐             ┌──────────────┐
   │  Vue 3 工作台  │◄──SSE───►│  Go 后端 8000  │◄──────────► │  SQLite (pure)│
   │  三栏·i18n   │           │  Workbench+API │             │  零 cgo 依赖   │
   └──────────────┘           └───────┬────────┘             └──────────────┘
                                      │
                   ┌──────────────────┼──────────────────┐
                   ▼                  ▼                  ▼
            ┌─────────────┐  ┌──────────────┐  ┌──────────────────┐
            │ HarnessRuntime│  │  Validator   │  │   Skill Pack    │
            │  B0-B4/BVAR  │  │  Registry    │  │ literature_search│
            │  策略·预算   │  │  4类验证器    │  │  file_organizer  │
            └──────┬───────┘  └──────────────┘  └──────────────────┘
                   │
                   ▼
            ┌──────────────────────────────┐
            │       Provider 抽象层         │
            │  OpenAI 兼容 · DeepSeek 定制  │
            │  prefix cache · stale 剪枝   │
            └──────────────────────────────┘
```

---

## ✦ 为什么需要它

8B 参数的小模型无法靠"自觉"完成长程任务——它们需要**结构化的执行框架**。

| 问题 | 解决方案 |
|------|---------|
| 模型输出不稳定 | **类型化契约**（TaskContract / ActionContract）约束每一次输出 |
| 成本不可控 | **三维预算**（token / 成本 / 时间），超出即停止并定向恢复 |
| 无可信验证 | **四类验证器**（存在性 / JSON Schema / 引用完整性 / 预算），失败触发恢复 |
| 无法追溯 | **全程事件化 + SHA-256 内容寻址 + JSONL 回放**，每一次失败都成为可学习经验 |
| 多人冲突 | **账户级数据隔离**（account_id 强制过滤 + 404 不泄露边界） |

> **核心信条**：让有限算力下的普通人，也能拥有可靠、清醒、愿意承担细节工作的智能协作系统。

---

## ✦ 核心特性

<table>
<tr>
<td width="50%" valign="top">

### 🧠 多策略路由
**B0-B4 五种基线策略** + **BVAR**（预算感知验证器路由）

- B0 固定单流程
- B1 单 Agent 闭环
- B2 串行角色复用
- B3 固定分支候选
- B4 BVAR 预算感知验证器路由

验证器驱动预算分配，失败自动恢复。

</td>
<td width="50%" valign="top">

### 🔒 受限工具执行
**权限治理 + 高危操作审批**

- 工具白名单
- 路径隔离
- 网络域白名单
- 风险审批门
- 默认拒绝

真实副作用必须经 Schema 校验、最小权限、预条件与人工审批。

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 🏢 多用户隔离
**账户级数据隔离**

- PBKDF2-HMAC-SHA256 密码哈希（210,000 迭代）
- HttpOnly SameSite=Strict Cookie
- account_id 强制过滤
- 跨账户访问返回 404（不泄露存在性）
- 公共模型池：system scope 供应商共享，用量归属调用方

</td>
<td width="50%" valign="top">

### 📊 Token 计量
**usage_events 落账 + 四窗口聚合**

- 按账户 / 供应商 / 时间维度聚合
- 四窗口实时计算
- 公共模型池成本视图
- 前端用量面板可视化

</td>
</tr>
<tr>
<td width="50%" valign="top">

### 🛠 供应商抽象
**25+ 预设供应商一键添加**

DeepSeek · Kimi · SiliconFlow · ModelScope · 本地 等

- OpenAI 兼容 Provider
- DeepSeek 定制（prefix cache 稳定会话 + stale 输出剪枝）
- 健康检查 + 激活切换
- 预设列表 + 自定义创建

</td>
<td width="50%" valign="top">

### 🧩 Skill 体系
**领域能力插件化**

- `literature_search`：四来源检索 · 去重 · 证据核验 · 四种导出
- `file_organizer`：回归样例 Skill
- 注册表 + manifest 声明
- 领域无关核心 + 领域 Skill 适配器

</td>
</tr>
</table>

---

## ✦ 技术栈

| 层 | 选型 | 约束 |
|----|------|------|
| **后端** | Go 1.26 | 唯一主线，`CGO_ENABLED=0` 单静态二进制 |
| **前端** | Vue 3 + TypeScript + Vite | 构建后 `embed.FS` 嵌入二进制 |
| **存储** | SQLite | 纯 Go 驱动 `modernc.org/sqlite`，零 cgo |
| **密码学** | 标准库 `crypto/pbkdf2` | 210,000 迭代 · 随机盐 · 恒定时间比较 |
| **分发** | 单文件交付 | 无运行时依赖，零安装 |

> **零外部依赖**：除标准库 + SQLite 驱动外，仅 uuid / humanize 等轻量工具库。

---

## ✦ 平台支持

```
  OS \ ARCH    amd64    arm64    loong64    riscv64
  ───────────────────────────────────────────────────
  Windows       ✅       ✅       —          —
  Linux         ✅       ✅       ✅         ✅
  macOS         ✅       ✅       —          —
```

> 面向国产化与新兴架构：**龙芯 loong64** 与 **RISC-V** 原生支持。
> 8 组合交叉编译，单静态二进制覆盖全平台。

---

## ✦ 快速开始

### 一键启动开发模式

```powershell
# Windows PowerShell 7+
.\dev.ps1
```

脚本自动完成：前端依赖检查 → 启动 Go 后端 → 健康检查就绪 → 启动 Vite 前端。

```
  ╔═══════════════════════════════════════════════════╗
  ║  浏览器打开: http://127.0.0.1:48623                ║
  ║  后端 API:  http://127.0.0.1:8000                 ║
  ║  按 Ctrl+C 停止所有服务                             ║
  ╚═══════════════════════════════════════════════════╝
```

### 手动启动

```bash
# 后端
go run ./cmd/server/ -addr 127.0.0.1:8000

# 前端（另一终端）
cd web && npm install && npm run dev
```

### 生产构建

```bash
make all            # 前端构建 + 本平台后端
make cross-compile  # 8 平台/架构组合交叉编译（CGO_ENABLED=0）
make test           # 全量测试
```

---

## ✦ 项目结构

```
cmd/server/              ── 服务入口：装配存储/运行时/鉴权/静态嵌入
internal/
├── harness/             ── 领域无关核心
│   ├── contract.go        ── 类型化契约（TaskContract / ActionContract）
│   ├── state.go           ── 状态机 + 版本守卫
│   ├── budget.go          ── 三维预算控制
│   ├── validator_*.go     ── 四类验证器 + 注册表聚合
│   ├── runtime.go         ── HarnessRuntime 执行循环
│   └── strategy_b*.go     ── B0-B4 + BVAR 策略
├── providers/           ── 供应商抽象层
│   ├── openai.go           ── OpenAI 兼容 Provider
│   ├── deepseek.go         ── DeepSeek 定制（prefix cache + stale 剪枝）
│   └── presets.go          ── 25+ 预设供应商列表
├── skills/              ── Skill 注册表 + 适配器
│   ├── registry.go         ── Skill 注册与发现
│   ├── literature_search/  ── 四来源检索 · 去重 · 证据核验 · 导出
│   └── file_organizer/     ── 回归样例 Skill
└── workbench/           ── 应用层
    ├── server.go           ── HTTP 路由 + 中间件
    ├── accounts.go         ── 多用户账户 + 密码哈希
    ├── session.go          ── 会话管理（HttpOnly Cookie）
    ├── auth_middleware.go   ── 鉴权 + account_id 注入
    ├── providers.go        ── 供应商档案 CRUD + 健康检查
    ├── usage.go            ── Token 计量 + 四窗口聚合
    └── task_drafts.go      ── 任务草案审批 → 运行时执行
web/
├── src/
│   ├── views/             ── LoginView + 工作台
│   ├── components/        ── 三栏布局 · 供应商面板 · 用量面板
│   ├── stores/            ── 响应式状态管理
│   ├── api/               ── 类型化 API 客户端
│   └── i18n/              ── 中 / 英双语
└── dist/                  ── 构建产物（embed.FS 嵌入二进制）
```

---

## ✦ 运行约束（设计边界）

- 强模型仅用于离线 `GoldReferencePack`，**不进入日常交互运行时**
- 不保存、不训练隐藏推理链；只记录计划、动作、观察、工件、验证与错误码
- 真实副作用必须经 **Schema 校验 + 最小权限 + 预条件 + 人工审批**
- 经验先回放、后入库：成功/失败轨迹通过固定环境回放才能成为 `skill_card`

---

## ✦ 研究问题

本项目旨在回答一个中心问题：**良好的 Harness、结构化通信、工件记忆、验证反馈和动态计算分配，能够将 12B 以下模型的任务级能力推进到什么程度？**

| ID | 研究问题 |
|----|---------|
| RQ1 | Harness 能为固定 8B 模型带来多少任务增益？ |
| RQ2 | 哪些任务值得触发多 Agent 协作？ |
| RQ3 | 如何在固定预算内分配测试时计算？ |
| RQ4 | 外部技能与记忆能补偿多少模型能力？ |
| RQ5 | 强教师经验如何迁移到小模型系统？ |
| RQ6 | 系统如何在风险约束下稳定执行？ |

---

## ✦ 许可证

**Apache License 2.0** —— 详见 [LICENSE](LICENSE) 与 [NOTICE](NOTICE)。

---

<div align="center">

**oh-my-loongAgent** · 让小模型做大事

`⚡ Go · Vue · SQLite · 零 cgo · 全平台 · 可审计 ⚡`

</div>