package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"

	_ "modernc.org/sqlite"
)

type Article struct {
	ID      int
	Title   string
	Content string
}

type UI struct {
	theme        *material.Theme
	idInput      widget.Editor
	queryBtn     widget.Clickable
	titleInput   widget.Editor
	contentInput widget.Editor
	saveBtn      widget.Clickable
	db           *sql.DB
	errorMsg     string
}

func getArticleByID(db *sql.DB, id int) (*Article, error) {
	var article Article
	query := "SELECT id, title, content FROM articles WHERE id = ?"
	err := db.QueryRow(query, id).Scan(&article.ID, &article.Title, &article.Content)
	if err != nil {
		return nil, err
	}
	return &article, nil
}

func updateArticleByID(db *sql.DB, id int, title, content string) error {
	query := "UPDATE articles SET title = ?, content = ? WHERE id = ?"
	_, err := db.Exec(query, title, content, id)
	return err
}

func main() {
	db, err := sql.Open("sqlite", "./wherite.sqlite3")
	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	go func() {
		w := new(app.Window)
		w.Option(
			app.Title("Wherite - 文章查询系统"),
			app.Size(unit.Dp(1000), unit.Dp(800)),
		)

		ui := &UI{
			theme: material.NewTheme(),
			db:    db,
		}
		ui.idInput.SingleLine = true
		ui.idInput.Submit = true
		ui.titleInput.SingleLine = true
		ui.contentInput.Submit = true

		if err := ui.Run(w); err != nil {
			log.Fatal(err)
		}
	}()

	app.Main()
}

func (ui *UI) Run(w *app.Window) error {
	var ops op.Ops

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			ui.Layout(gtx, w)

			e.Frame(gtx.Ops)
		}
	}
}

func (ui *UI) Layout(gtx layout.Context, w *app.Window) {
	if event, ok := ui.idInput.Update(gtx); ok {
		if _, ok := event.(widget.SubmitEvent); ok {
			ui.queryArticle()
		}
	}

	if ui.queryBtn.Clicked(gtx) {
		ui.queryArticle()
	}

	if ui.saveBtn.Clicked(gtx) {
		ui.saveArticle()
	}

	layout.Flex{
		Axis:      layout.Vertical,
		Alignment: layout.Middle,
	}.Layout(gtx,
		// ID输入和查询按钮区域
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis:      layout.Horizontal,
					Alignment: layout.Middle,
				}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Editor(ui.theme, &ui.idInput, "请输入文章ID").Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.queryBtn, "查询")
							btn.TextSize = unit.Sp(16)
							return btn.Layout(gtx)
						})
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							saveBtn := material.Button(ui.theme, &ui.saveBtn, "保存")
							saveBtn.TextSize = unit.Sp(16)
							return saveBtn.Layout(gtx)
						})
					}),
				)
			})
		}),
		// 错误消息显示
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.errorMsg != "" {
				return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(ui.theme, unit.Sp(16), ui.errorMsg)
					lbl.Color = ui.theme.ContrastBg
					return lbl.Layout(gtx)
				})
			}
			return layout.Dimensions{}
		}),
		// 标题输入框
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(ui.theme, unit.Sp(18), "标题:")
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						titleEditor := material.Editor(ui.theme, &ui.titleInput, "文章标题")
						titleEditor.TextSize = unit.Sp(20)
						titleEditor.Font.Weight = font.Bold
						return titleEditor.Layout(gtx)
					}),
				)
			})
		}),
		// 内容输入框
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(ui.theme, unit.Sp(18), "内容:")
						return lbl.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						contentEditor := material.Editor(ui.theme, &ui.contentInput, "文章内容")
						contentEditor.TextSize = unit.Sp(16)
						return contentEditor.Layout(gtx)
					}),
				)
			})
		}),
	)
}

func (ui *UI) queryArticle() {
	idStr := ui.idInput.Text()
	if idStr == "" {
		ui.errorMsg = "请输入文章ID"
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.errorMsg = "无效的ID格式，请输入数字"
		return
	}

	article, err := getArticleByID(ui.db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			ui.errorMsg = "未找到该文章"
		} else {
			ui.errorMsg = fmt.Sprintf("查询失败: %v", err)
		}
		return
	}

	// 将查询结果填充到输入框中
	ui.titleInput.SetText(article.Title)
	ui.contentInput.SetText(article.Content)
	ui.errorMsg = ""
}

func (ui *UI) saveArticle() {
	idStr := ui.idInput.Text()
	if idStr == "" {
		ui.errorMsg = "请先输入文章ID并查询"
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		ui.errorMsg = "无效的ID格式，请输入数字"
		return
	}

	title := ui.titleInput.Text()
	content := ui.contentInput.Text()

	if title == "" {
		ui.errorMsg = "标题不能为空"
		return
	}

	if content == "" {
		ui.errorMsg = "内容不能为空"
		return
	}

	err = updateArticleByID(ui.db, id, title, content)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("保存失败: %v", err)
		return
	}

	ui.errorMsg = "保存成功！"
}
