# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with this repository.

## Build & Run

```bash
# Build executable
go build -o wherite

# Run directly
go run .
```

## Architecture

Three-module structure:

- **[wherite_main.go](wherite_main.go)** - Entry point. Initializes database connection and starts Gio event loop in a goroutine.
- **[wherite_gui.go](wherite_gui.go)** - UI layer using Gio. Manages `UI` struct with input fields, buttons, and rendering via `Layout()` method. Calls database functions for query/save operations.
- **[wherite_database.go](wherite_database.go)** - Data access layer. Provides `Article` struct and functions for CRUD operations including `SearchArticles` and `RenameArticleByID`. Uses SQLite triggers for automatic timestamp management.
- **[wherite_markdown.go](wherite_markdown.go)** - Markdown parsing using goldmark. Converts Markdown to HTML and provides block-level parsing for preview rendering.
- **[wherite_syntax.go](wherite_syntax.go)** - Syntax highlighting tokenization for Markdown elements (headings, bold, italic, code, links, quotes).

Flow: `main()` → connects to SQLite → launches GUI goroutine → UI calls database functions for article operations.

## Tech Stack

- **Go 1.25.5** with standard `database/sql`
- **Gio v0.9.0** for cross-platform GUI (pure Go, no CGO)
- **SQLite** via `modernc.org/sqlite` (no GCC required)
- **Goldmark** for Markdown parsing (GFM, tables, strikethrough, task lists)
