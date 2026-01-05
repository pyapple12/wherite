package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// Article 表示数据库中的文章记录
type Article struct {
	ID        int
	Title     string
	Content   string
	CreatedAt string
	UpdatedAt string
}

// ConnectDB 连接SQLite数据库并返回数据库连接
func ConnectDB(dataSource string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dataSource)
	if err != nil {
		return nil, fmt.Errorf("无法连接数据库: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}

	return db, nil
}

// CloseDB 关闭数据库连接
func CloseDB(db *sql.DB) {
	if db != nil {
		db.Close()
	}
}

// GetArticleByID 根据ID查询文章
func GetArticleByID(db *sql.DB, id int) (*Article, error) {
	var article Article
	query := "SELECT id, title, content, created_at, updated_at FROM articles WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Content, &article.CreatedAt, &article.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("未找到ID为%d的文章", id)
		}
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	return &article, nil
}

// UpdateArticleByID 根据ID更新文章
func UpdateArticleByID(db *sql.DB, id int, title, content string) error {
	query := "UPDATE articles SET title = ?, content = ? WHERE id = ?"
	result, err := db.Exec(query, title, content, id)
	if err != nil {
		return fmt.Errorf("更新失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("未找到ID为%d的文章", id)
	}

	return nil
}

// CreateArticle 创建新文章
func CreateArticle(db *sql.DB, title, content string) (int64, error) {
	query := "INSERT INTO articles (title, content) VALUES (?, ?)"
	result, err := db.Exec(query, title, content)
	if err != nil {
		return 0, fmt.Errorf("创建文章失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取文章ID失败: %w", err)
	}

	return id, nil
}

// DeleteArticleByID 根据ID删除文章
func DeleteArticleByID(db *sql.DB, id int) error {
	query := "DELETE FROM articles WHERE id = ?"
	result, err := db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("删除文章失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("未找到ID为%d的文章", id)
	}

	return nil
}

// GetAllArticles 获取所有文章
func GetAllArticles(db *sql.DB) ([]Article, error) {
	query := "SELECT id, title, content, created_at, updated_at FROM articles ORDER BY id"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.Title, &article.Content, &article.CreatedAt, &article.UpdatedAt); err != nil {
			return nil, fmt.Errorf("读取数据失败: %w", err)
		}
		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历数据失败: %w", err)
	}

	return articles, nil
}

// InitializeDatabase 初始化数据库表
func InitializeDatabase(db *sql.DB) error {
	// 创建表（如果不存在）
	createTable := `
	CREATE TABLE IF NOT EXISTS articles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TEXT DEFAULT (datetime('now', 'localtime')),
		updated_at TEXT DEFAULT (datetime('now', 'localtime'))
	)
	`
	_, err := db.Exec(createTable)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	// 添加时间戳列（如果表中不存在）
	alterTable := `
	ALTER TABLE articles ADD COLUMN created_at TEXT DEFAULT (datetime('now', 'localtime'))
	`
	db.Exec(alterTable)

	alterTable = `
	ALTER TABLE articles ADD COLUMN updated_at TEXT DEFAULT (datetime('now', 'localtime'))
	`
	db.Exec(alterTable)

	// 创建 INSERT 触发器 - 自动设置创建时间
	createInsertTrigger := `
	CREATE TRIGGER IF NOT EXISTS set_created_at
	AFTER INSERT ON articles
	BEGIN
		UPDATE articles SET created_at = datetime('now', 'localtime'), updated_at = datetime('now', 'localtime') WHERE id = NEW.id;
	END
	`
	_, err = db.Exec(createInsertTrigger)
	if err != nil {
		return fmt.Errorf("创建INSERT触发器失败: %w", err)
	}

	// 创建 UPDATE 触发器 - 自动更新修改时间
	createUpdateTrigger := `
	CREATE TRIGGER IF NOT EXISTS set_updated_at
	AFTER UPDATE ON articles
	BEGIN
		UPDATE articles SET updated_at = datetime('now', 'localtime') WHERE id = NEW.id;
	END
	`
	_, err = db.Exec(createUpdateTrigger)
	if err != nil {
		return fmt.Errorf("创建UPDATE触发器失败: %w", err)
	}

	return nil
}

// PrintDatabaseInfo 打印数据库信息
func PrintDatabaseInfo(db *sql.DB) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM articles").Scan(&count)
	if err != nil {
		log.Printf("获取文章数量失败: %v", err)
		return
	}
	log.Printf("数据库已连接，当前有 %d 篇文章", count)
}
