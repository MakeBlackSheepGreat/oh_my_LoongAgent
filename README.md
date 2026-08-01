# SLIM-AGENT · 小型模型多 Agent 编排引擎

> **面向 ≤12B 参数本地模型的通用 Agent Harness**
> 以类型化契约约束模型输出，用验证器、预算与审计构建可信的执行闭环。

```
┌────────────────────────────────────────────────────────────────────┐
│  TaskContract ──▶ HarnessRuntime ──▶ Skill Pack ──▶ typed artifacts │
│                        │                 │                │        │
│                        ▼                 ▼                ▼        │
│                  B0-B4 / BVAR       tools          validators      │
│                        └──────────────┬─────────────────┘          │
│                                       ▼                            │
│                    SQLite + SHA-256 + JSONL replay（可回放可审计）     │
└────────────────────────────────────────────────────────────────────┘
```

## 为什么需要它

8B 参数的小模型无法靠"自觉"完成长程任务——它们需要**结构化的执行框架**：

- 模型输出被约束为类型化 `TaskContract` / `ActionContract`，而非自由文本
- 每个动作受**预算**（token/成本/时间）与**权限**（工具治理）双重约束
- 每个结果过**验证器**（存在性/JSON Schema/引用完整性/预算）
- 全程事件化、内容寻址（SHA-256）、可回放——**每一次失败都成为可学习的经验**

## 核心特性

| 能力 | 说明 |
| --- | --- |
| 🧠 **多策略路由** | B0-B4 五种基线策略 + BVAR（预算感知验证器路由），验证器驱动预算分配 |
| 🔒 **受限工具执行** | 权限治理 + 高危操作审批，默认拒绝 |
| 🧮 **预算控制** | token / 成本 / 时间三维预算，超出即停止并定向恢复 |
| ✅ **验证器体系** | 内建四类验证器 + 注册表聚合，失败触发恢复（retry/repair/human_review） |
| 🏢 **多用户隔离** | 账户级数据隔离（account_id 强制过滤）、会话管理、404 不泄露边界 |
| 📊 **Token 计量** | usage_events 落账、四窗口聚合、公共模型池视图 |
| 🌐 **多语言** | 中 / 英 i18n，偏好持久化到账户 |
| 🔑 **安全认证** | PBKDF2-HMAC-SHA256 密码哈希、HttpOnly Cookie、防枚举与时序侧信道防护 |
| 🧩 **Skill 体系** | 领域能力插件化：literature_search（四来源检索/去重/证据核验/导出）、file_organizer 回归样例 |
| 🛠 **供应商抽象** | OpenAI 兼容 Provider + DeepSeek 定制（prefix cache、stale 输出剪枝）、预设档案一键添加 |

## 技术栈

| 层 | 选型 |
| --- | --- |
| 后端 | Go 1.26（唯一主线，`CGO_ENABLED=0` 单静态二进制） |
| 前端 | Vue 3 + TypeScript + Vite（构建后 embed.FS 嵌入二进制） |
| 存储 | SQLite（纯 Go 驱动 `modernc.org/sqlite`，零 cgo） |
| 密码学 | 标准库 `crypto/pbkdf2`（210,000 迭代 · 随机盐 · 恒定时间比较） |
| 分发 | 单文件交付，无运行时依赖 |

## 平台支持

| OS \ ARCH | amd64 | arm64 | loong64 | riscv64 |
| --- | --- | --- | --- | --- |
| Windows | ✅ | ✅ | — | — |
| Linux | ✅ | ✅ | ✅ | ✅ |
| macOS | ✅ | ✅ | — | — |

> 面向国产化与新兴架构：龙芯（loong64）与 RISC-V 原生支持。

## 快速开始

### 一键启动开发模式（Windows）

```powershell
.\dev.ps1
```

脚本自动完成：依赖检查 → 启动 Go 后端（`127.0.0.1:8000`）→ 健康检查就绪 → 启动 Vite 前端（`http://127.0.0.1:48623`）。Ctrl+C 停止全部服务（推荐 PowerShell 7+）。

### 手动启动

```bash
# 后端（flag 形式，无子命令）
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

## 项目结构

```
cmd/server/           服务入口：装配存储/运行时/鉴权/静态嵌入
internal/
├── harness/          领域无关核心：契约、状态机、预算、验证器、B0-B4/BVAR 策略、运行时
├── providers/        OpenAI 兼容 Provider 与 DeepSeek 定制（prefix cache、stale 剪枝）
├── skills/           Skill 注册表 + literature_search / file_organizer
└── workbench/        应用层：多用户账户、会话、数据隔离、任务草案审批、Token 计量
web/                  Vue 3 工作台（三栏布局 + SSE 实时事件 + i18n）
```

## 运行约束（设计边界）

- 强模型仅用于离线 `GoldReferencePack`，不进入日常交互运行时
- 不保存、不训练隐藏推理链；只记录计划、动作、观察、工件、验证与错误码
- 真实副作用必须经 Schema 校验、最小权限、预条件与人工审批
- 经验先回放、后入库：成功/失败轨迹通过固定环境回放才能成为 skill_card

## 许可证

Apache License 2.0 —— 详见 [LICENSE](LICENSE) 与 [NOTICE](NOTICE)。
