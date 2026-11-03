# AGENTS.md

## 项目简介

象映笔记（Xiangying Notes）是一款基于 **Go + MyGo 原生自绘 UI** 的现代化桌面 Markdown 笔记软件。
彻底告别复杂的 Node.js/Bun/Browser 运行时，采用纯 Go 全栈开发，GPU 硬件加速原生渲染，提供毫秒级即时启动与丝滑编辑体验。

## 技术栈与架构

- **语言**: Go 1.27+
- **GUI 框架**: [MyGo](https://github.com/egoist/mygo) (`github.com/egoist/mygo` 及 `github.com/egoist/mygo/ui`)
- **富文本与 Markdown**: `github.com/egoist/mygo/ui/richtext`
- **本地存储**: 纯 Go 嵌入式 SQLite (`modernc.org/sqlite`)，无需 CGO

## 代码目录结构

- `main.go`：桌面应用入口与窗口生命周期管理
- `assets/`：应用图标及内置 SVG 矢量图标
- `internal/model/`：领域数据结构（Notebook、Note、TagCount、TaskItem、HeadingItem 等）
- `internal/db/`：SQLite 存储与迁移（笔记本、笔记、标签、待办及全文过滤）
- `internal/markdown/`：Markdown 解析器（标签提取、待办解析、大纲标题提取与摘要提炼）
- `internal/ui/`：MyGo 原生 UI 组件（三栏响应式工作区、侧边栏、笔记列表、富文本编辑器、待办中心与弹窗）

## 开发与验证指令

```bash
# 格式化代码
gofmt -w .

# 运行单元与集成测试（包含 ui.NewTester 无窗口 UI 模拟测试）
go test -v ./...

# 编译桌面可执行文件
go build -o xiangying.exe .
```
