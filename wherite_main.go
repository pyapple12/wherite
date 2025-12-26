package main

import (
	"log"

	"gioui.org/app"
	"gioui.org/unit"
)

func main() {
	// 连接数据库
	db, err := ConnectDB("./wherite.sqlite3")
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}
	defer CloseDB(db)

	// 初始化数据库表（如果不存在）
	err = InitializeDatabase(db)
	if err != nil {
		log.Printf("警告: 初始化数据库表失败: %v", err)
	}

	// 打印数据库信息
	PrintDatabaseInfo(db)

	// 启动图形界面
	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Wherite - 文章查询系统"),
			app.Size(unit.Dp(1000), unit.Dp(800)),
		)

		ui := NewUI(db)

		if err := ui.Run(w); err != nil {
			log.Fatal(err)
		}
	}()

	// 运行主事件循环
	app.Main()
}
