# task0：项目指导文件与技术栈冻结

**状态**：completed  
**目标**：冻结 Go + Vue 3 + SQLite + 多平台多架构的技术栈边界，确认指导文件与决策记录就绪。

## 输入

- 用户决定：Go 唯一后端、Vue 3 + TypeScript 唯一前端、删除 Python 不存档、多平台多架构支持。
- 参考来源：DeepSeek-Reasonix（MIT，C-007）。

## 输出

- RULES.md v0.3：写入 Go/Vue 唯一主线、Windows/Linux/macOS × amd64/arm64/loong64/riscv64、CGO_ENABLED=0 单静态二进制、禁止 cgo。
- D-013、D-014：Go 重写与删除 Python 的决策记录。
- C-007：Reasonix 架构参考来源登记。
- ATTRIBUTIONS.md C-004 标记 retired。
- Python 代码与 React 前端已删除。
- Vue 3 + TypeScript 骨架已建（`web/`）。
- Go module 已初始化（`go.mod`，module `slim-agent`，Go 1.26.5）。

## 验收条件

1. RULES.md 明确 Go+Vue 唯一主线与多平台多架构要求。
2. D-013、D-014、C-007 可追溯。
3. 项目内无 Python 代码进入构建/测试/运行路径。
4. `web/` 可 `npm run build` 通过。

## 验证证据

- RULES.md v0.3 已写入全部硬约束。
- D-013（Go 重写决策）、D-014（删除 Python、切换 Vue、多平台多架构）已记录。
- C-007（Reasonix MIT 来源）已登记。
- Python 代码全部删除（`harness/`、`workbench/`、`skills/`、`tests/`、`bvar/`、`pyproject.toml` 等）。
- Vue 骨架 `npm run build` 通过（212ms）。
- Go 1.26.5 安装就绪，`go.mod` 初始化。
