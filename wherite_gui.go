package main

import (
	"database/sql"
	"fmt"
	"strconv"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// UI 结构体管理图形界面状态和组件
type UI struct {
	theme        *material.Theme
	idInput      widget.Editor
	queryBtn     widget.Clickable
	createBtn    widget.Clickable
	titleInput   widget.Editor
	contentInput widget.Editor
	saveBtn      widget.Clickable
	db           *sql.DB
	errorMsg     string
	isCreating   bool
}

// NewUI 创建一个新的UI实例
func NewUI(db *sql.DB) *UI {
	ui := &UI{
		theme: material.NewTheme(),
		db:    db,
	}
	ui.idInput.SingleLine = true
	ui.idInput.Submit = true
	ui.titleInput.SingleLine = true
	ui.contentInput.Submit = true
	return ui
}

// Run 运行图形界面事件循环
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

// Layout 渲染图形界面布局
func (ui *UI) Layout(gtx layout.Context, w *app.Window) {
	// 处理编辑器事件
	if event, ok := ui.idInput.Update(gtx); ok {
		if _, ok := event.(widget.SubmitEvent); ok {
			ui.queryArticle()
		}
	}

	// 处理按钮点击事件
	if ui.queryBtn.Clicked(gtx) {
		ui.queryArticle()
	}

	if ui.createBtn.Clicked(gtx) {
		ui.createArticle()
	}

	if ui.saveBtn.Clicked(gtx) {
		ui.saveArticle()
	}

	// 整体布局
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
							createBtn := material.Button(ui.theme, &ui.createBtn, "新建")
							createBtn.TextSize = unit.Sp(16)
							return createBtn.Layout(gtx)
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

// queryArticle 查询文章
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

	article, err := GetArticleByID(ui.db, id)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("查询失败: %v", err)
		return
	}

	// 将查询结果填充到输入框中
	ui.titleInput.SetText(article.Title)
	ui.contentInput.SetText(article.Content)
	ui.errorMsg = ""
}

// createArticle 进入新建模式，清空输入框
func (ui *UI) createArticle() {
	ui.idInput.SetText("")
	ui.titleInput.SetText("")
	ui.contentInput.SetText("")
	ui.errorMsg = ""
	ui.isCreating = true
}

// saveArticle 保存文章
func (ui *UI) saveArticle() {
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

	if ui.isCreating {
		// 新建模式：创建新文章
		id, err := CreateArticle(ui.db, title, content)
		if err != nil {
			ui.errorMsg = fmt.Sprintf("创建失败: %v", err)
			return
		}
		ui.idInput.SetText(strconv.FormatInt(id, 10))
		ui.isCreating = false
		ui.errorMsg = "创建成功！"
	} else {
		// 编辑模式：更新现有文章
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

		err = UpdateArticleByID(ui.db, id, title, content)
		if err != nil {
			ui.errorMsg = fmt.Sprintf("保存失败: %v", err)
			return
		}
		ui.errorMsg = "保存成功！"
	}
}
