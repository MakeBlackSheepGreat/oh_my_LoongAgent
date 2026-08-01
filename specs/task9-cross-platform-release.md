# task9：多平台多架构与发布验证

**状态**：in_progress（子任务 task9.0 待开始；task9.1 前端产物已存在待复验；task9.2-task9.5 未开始）  
**依赖**：task0-task10 全部完成（task10 已于 v1.8 完成）  
**输入**：现有 Go 后端 + Vue 3 前端代码库  
**输出**：8 组合交叉编译通过 + CI 配置 + 发布检查清单 + 健康检查定时任务

---

## 目标

确保项目可以从零构建为生产就绪的静态二进制，覆盖所有目标平台与架构，并建立可重复的发布流程。

## 子任务

### task9.0 交叉编译全组合验证

**验收条件**：
- `make cross-compile` 在 Windows/Linux/macOS 上对以下 8 组合全部编译通过：
  - windows/amd64, windows/arm64
  - linux/amd64, linux/arm64, linux/loong64, linux/riscv64
  - darwin/amd64, darwin/arm64
- 每个二进制均为 CGO_ENABLED=0 静态链接
- 产物输出到 `dist/slim-agent-{os}-{arch}[.exe]`

**验证证据**：
- `make cross-compile` 完整日志
- 各二进制 `file` 命令输出（或等效检查）

### task9.1 前端构建验证

**验收条件**：
- `make frontend` 完成 Vue 3 生产构建（`web/dist/`）
- `go build` 成功嵌入 `web/dist` 到二进制
- SPA fallback 路由正常工作

**验证证据**：
- `npm run build` 无错误
- `go build -o dist/slim-agent ./cmd/server/` 通过
- 启动后访问非 API 路径返回 index.html

### task9.2 健康检查定时任务

**验收条件**：
- 后端启动时自动对已激活的 provider 执行定时健康检查（每 5 分钟）
- 健康检查结果缓存到内存，供前端 `/api/providers/{id}/health` 快速返回
- 健康检查失败时通过 EventHub 广播 `provider:health_fail` 事件
- 前端 SSE 接收后显示通知

**验证证据**：
- 启动后等待 5 分钟，观察后端日志确认健康检查执行
- 手动关闭 provider 后，SSE 事件触发前端通知

### task9.3 CI 配置（GitHub Actions）

**验收条件**：
- `.github/workflows/ci.yml` 覆盖以下步骤：
  - 代码检出
  - Go 缓存恢复
  - `go vet ./...`
  - `go test -count=1 ./...`
  - 前端依赖安装 + `npm run build`
  - 至少 4 组合交叉编译（windows/amd64、linux/amd64、linux/arm64、darwin/arm64）
  - 产物上传为 workflow artifact

**验证证据**：
- CI 绿色运行截图或日志

### task9.4 发布检查清单

**验收条件**：
- `RELEASE_CHECKLIST.md` 文件列出以下步骤：
  - [ ] 版本号确认（`PROJECT_GOALS.md` 或 git tag）
  - [ ] `go vet ./...` 通过
  - [ ] `go test -count=1 ./...` 全部通过
  - [ ] 前端构建通过
  - [ ] 8 组合交叉编译全部通过
  - [ ] 启动后 API 健康检查返回 200
  - [ ] 注册、登录、项目/会话 CRUD 基本功能可用
  - [ ] 供应商预设列表加载正常
  - [ ] 已激活 provider 健康检查定时任务运行
  - [ ] 环境变量密钥设置正确

**验证证据**：
- 文件存在且步骤完整

### task9.5 测试覆盖验证

**验收条件**：
- `go test -count=1 ./...` 全部通过，无 skipped
- 测试覆盖率不低于 60%（语句覆盖率）

**验证证据**：
- `go test -count=1 -cover ./...` 输出

---

## 依赖关系

```
task9.0 → task9.1 → task9.3 → task9.4
                ↘
task9.2 → task9.4
                ↘
task9.5 → task9.4
```

## 风险

- loong64 和 riscv64 的交叉编译工具链可能需要手动安装
- `modernc.org/sqlite` 在这些架构上的兼容性需验证
- CI 无法直接运行需要 GPU 或专用硬件的测试