# CLAUDE.md

本文档为 Claude Code (claude.ai/code) 在本代码库中工作提供指导。

## 编译与运行

```bash
# 编译可执行文件
go build -o wherite

# 直接运行
go run .
```

## 架构

三模块结构：

- **[wherite_main.go](wherite_main.go)** - 程序入口。初始化数据库连接并在 goroutine 中启动 Gio 事件循环。
- **[wherite_gui.go](wherite_gui.go)** - UI 层，使用 Gio。通过 `Layout()` 方法管理 `UI` 结构体（包含输入框、按钮和渲染逻辑），调用数据库函数进行查询/保存操作。
- **[wherite_database.go](wherite_database.go)** - 数据访问层。提供 `Article` 结构体和 CRUD 操作函数（`GetArticleByID`、`UpdateArticleByID`、`CreateArticle`、`DeleteArticleByID`、`GetAllArticles`）。使用纯 Go SQLite 驱动 (`modernc.org/sqlite`)。

流程：`main()` → 连接 SQLite → 启动 GUI goroutine → UI 调用数据库函数进行文章操作。

## 技术栈

- **Go 1.25.4** + 标准库 `database/sql`
- **Gio v0.9.0** 跨平台 GUI 框架（纯 Go 实现，无需 CGO）
- **SQLite** 驱动 `modernc.org/sqlite`（无需 GCC）
