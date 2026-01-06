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

// Category 表示文章分类
type Category struct {
	ID   int
	Name string
}

// Tag 表示文章标签
type Tag struct {
	ID   int
	Name string
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

// RenameArticleByID 根据ID重命名文章
func RenameArticleByID(db *sql.DB, id int, newTitle string) error {
	query := "UPDATE articles SET title = ? WHERE id = ?"
	result, err := db.Exec(query, newTitle, id)
	if err != nil {
		return fmt.Errorf("重命名文章失败: %w", err)
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

// columnExists 检查表中是否存在指定列
func columnExists(db *sql.DB, tableName, columnName string) (bool, error) {
	query := `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`
	var count int
	err := db.QueryRow(query, tableName, columnName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
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

	// 创建分类表
	createCategoryTable := `
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	)
	`
	_, err = db.Exec(createCategoryTable)
	if err != nil {
		return fmt.Errorf("创建分类表失败: %w", err)
	}

	// 创建标签表
	createTagTable := `
	CREATE TABLE IF NOT EXISTS tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	)
	`
	_, err = db.Exec(createTagTable)
	if err != nil {
		return fmt.Errorf("创建标签表失败: %w", err)
	}

	// 创建文章分类关联表
	createArticleCategoryTable := `
	CREATE TABLE IF NOT EXISTS article_categories (
		article_id INTEGER NOT NULL,
		category_id INTEGER NOT NULL,
		PRIMARY KEY (article_id, category_id),
		FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
		FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
	)
	`
	_, err = db.Exec(createArticleCategoryTable)
	if err != nil {
		return fmt.Errorf("创建文章分类关联表失败: %w", err)
	}

	// 创建文章标签关联表
	createArticleTagTable := `
	CREATE TABLE IF NOT EXISTS article_tags (
		article_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (article_id, tag_id),
		FOREIGN KEY (article_id) REFERENCES articles(id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
	)
	`
	_, err = db.Exec(createArticleTagTable)
	if err != nil {
		return fmt.Errorf("创建文章标签关联表失败: %w", err)
	}

	// 检查并添加 created_at 列（如果不存在）
	exists, err := columnExists(db, "articles", "created_at")
	if err != nil {
		return fmt.Errorf("检查created_at列失败: %w", err)
	}
	if !exists {
		alterTable := `ALTER TABLE articles ADD COLUMN created_at TEXT DEFAULT (datetime('now', 'localtime'))`
		_, err = db.Exec(alterTable)
		if err != nil {
			return fmt.Errorf("添加created_at列失败: %w", err)
		}
	}

	// 检查并添加 updated_at 列（如果不存在）
	exists, err = columnExists(db, "articles", "updated_at")
	if err != nil {
		return fmt.Errorf("检查updated_at列失败: %w", err)
	}
	if !exists {
		alterTable := `ALTER TABLE articles ADD COLUMN updated_at TEXT DEFAULT (datetime('now', 'localtime'))`
		_, err = db.Exec(alterTable)
		if err != nil {
			return fmt.Errorf("添加updated_at列失败: %w", err)
		}
	}

	// 删除旧的触发器（如果存在）
	db.Exec("DROP TRIGGER IF EXISTS set_created_at")
	db.Exec("DROP TRIGGER IF EXISTS set_updated_at")

	// 创建 AFTER INSERT 触发器 - 确保时间戳被设置
	createInsertTrigger := `
	CREATE TRIGGER IF NOT EXISTS set_created_at
	AFTER INSERT ON articles
	WHEN NEW.created_at IS NULL OR NEW.updated_at IS NULL
	BEGIN
		UPDATE articles 
		SET created_at = COALESCE(NEW.created_at, datetime('now', 'localtime')),
		    updated_at = COALESCE(NEW.updated_at, datetime('now', 'localtime'))
		WHERE id = NEW.id;
	END
	`
	_, err = db.Exec(createInsertTrigger)
	if err != nil {
		return fmt.Errorf("创建INSERT触发器失败: %w", err)
	}

	// 创建 AFTER UPDATE 触发器 - 自动更新修改时间
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

// CreateCategory 创建分类
func CreateCategory(db *sql.DB, name string) (int64, error) {
	query := "INSERT INTO categories (name) VALUES (?)"
	result, err := db.Exec(query, name)
	if err != nil {
		return 0, fmt.Errorf("创建分类失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取分类ID失败: %w", err)
	}

	return id, nil
}

// GetAllCategories 获取所有分类
func GetAllCategories(db *sql.DB) ([]Category, error) {
	query := "SELECT id, name FROM categories ORDER BY name"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询分类失败: %w", err)
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.ID, &category.Name); err != nil {
			return nil, fmt.Errorf("读取分类数据失败: %w", err)
		}
		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分类数据失败: %w", err)
	}

	return categories, nil
}

// CreateTag 创建标签
func CreateTag(db *sql.DB, name string) (int64, error) {
	query := "INSERT INTO tags (name) VALUES (?)"
	result, err := db.Exec(query, name)
	if err != nil {
		return 0, fmt.Errorf("创建标签失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("获取标签ID失败: %w", err)
	}

	return id, nil
}

// GetAllTags 获取所有标签
func GetAllTags(db *sql.DB) ([]Tag, error) {
	query := "SELECT id, name FROM tags ORDER BY name"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询标签失败: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("读取标签数据失败: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历标签数据失败: %w", err)
	}

	return tags, nil
}

// SetArticleCategory 设置文章分类
func SetArticleCategory(db *sql.DB, articleID, categoryID int) error {
	query := "INSERT OR REPLACE INTO article_categories (article_id, category_id) VALUES (?, ?)"
	_, err := db.Exec(query, articleID, categoryID)
	if err != nil {
		return fmt.Errorf("设置文章分类失败: %w", err)
	}
	return nil
}

// GetArticleCategory 获取文章分类
func GetArticleCategory(db *sql.DB, articleID int) (*Category, error) {
	query := `
		SELECT c.id, c.name 
		FROM categories c
		INNER JOIN article_categories ac ON c.id = ac.category_id
		WHERE ac.article_id = ?
	`
	var category Category
	err := db.QueryRow(query, articleID).Scan(&category.ID, &category.Name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("获取文章分类失败: %w", err)
	}
	return &category, nil
}

// AddArticleTag 为文章添加标签
func AddArticleTag(db *sql.DB, articleID, tagID int) error {
	query := "INSERT OR IGNORE INTO article_tags (article_id, tag_id) VALUES (?, ?)"
	_, err := db.Exec(query, articleID, tagID)
	if err != nil {
		return fmt.Errorf("添加文章标签失败: %w", err)
	}
	return nil
}

// RemoveArticleTag 移除文章标签
func RemoveArticleTag(db *sql.DB, articleID, tagID int) error {
	query := "DELETE FROM article_tags WHERE article_id = ? AND tag_id = ?"
	_, err := db.Exec(query, articleID, tagID)
	if err != nil {
		return fmt.Errorf("移除文章标签失败: %w", err)
	}
	return nil
}

// GetArticleTags 获取文章的所有标签
func GetArticleTags(db *sql.DB, articleID int) ([]Tag, error) {
	query := `
		SELECT t.id, t.name 
		FROM tags t
		INNER JOIN article_tags at ON t.id = at.tag_id
		WHERE at.article_id = ?
		ORDER BY t.name
	`
	rows, err := db.Query(query, articleID)
	if err != nil {
		return nil, fmt.Errorf("查询文章标签失败: %w", err)
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, fmt.Errorf("读取标签数据失败: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历标签数据失败: %w", err)
	}

	return tags, nil
}

// SearchArticles 搜索文章
func SearchArticles(db *sql.DB, keyword string) ([]Article, error) {
	query := `
		SELECT id, title, content, created_at, updated_at 
		FROM articles 
		WHERE title LIKE ? OR content LIKE ?
		ORDER BY id
	`
	searchPattern := "%" + keyword + "%"
	rows, err := db.Query(query, searchPattern, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("搜索文章失败: %w", err)
	}
	defer rows.Close()

	var articles []Article
	for rows.Next() {
		var article Article
		if err := rows.Scan(&article.ID, &article.Title, &article.Content, &article.CreatedAt, &article.UpdatedAt); err != nil {
			return nil, fmt.Errorf("读取文章数据失败: %w", err)
		}
		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历文章数据失败: %w", err)
	}

	return articles, nil
}
