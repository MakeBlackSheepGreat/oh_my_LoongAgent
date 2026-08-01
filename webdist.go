// Package webdist 嵌入 Vue 前端构建产物（web/dist）。
// embed 路径相对本文件所在目录（项目根），故置于项目根。
package webdist

import "embed"

// FS 嵌入 web/dist 全部文件（all: 前缀包含 _ 开头文件如 assets/）。
// web/dist 缺失时 go build 失败；Makefile 的 `make all` 先执行 `make frontend`。
//
//go:embed all:web/dist
var FS embed.FS
