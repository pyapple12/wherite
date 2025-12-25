package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	_ "modernc.org/sqlite"
)

// Article 定义文章结构体
type Article struct {
	ID      int
	Title   string
	Content string
}

// getArticleByID 根据id查询文章
func getArticleByID(db *sql.DB, id int) (*Article, error) {
	var article Article
	query := "SELECT id, title, content FROM articles WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Content)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func main() {
	// 连接数据库
	db, err := sql.Open("sqlite", "./wherite.sqlite3")
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}
	defer db.Close()

	// 测试数据库连接
	err = db.Ping()
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	// 命令行界面
	fmt.Println("文章查询系统")
	fmt.Println("请输入文章ID (输入0退出):")

	for {
		var input string
		fmt.Print("> ")
		_, err := fmt.Scanln(&input)
		if err != nil {
			log.Printf("输入错误: %v", err)
			continue
		}

		id, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("无效的ID格式，请输入数字。")
			continue
		}

		if id == 0 {
			fmt.Println("退出系统。")
			os.Exit(0)
		}

		article, err := getArticleByID(db, id)
		if err != nil {
			if err == sql.ErrNoRows {
				fmt.Println("未找到该文章。")
			} else {
				log.Printf("查询失败: %v", err)
				fmt.Println("查询失败。")
			}
			continue
		}

		// 显示文章
		fmt.Println("\n=== 文章内容 ===")
		fmt.Printf("ID: %d\n", article.ID)
		fmt.Printf("标题: %s\n", article.Title)
		fmt.Printf("内容: %s\n", article.Content)
		fmt.Println("================")

		fmt.Println("请输入下一个文章ID (输入0退出):")
	}
}
