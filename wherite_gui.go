package main

import (
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"strconv"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// UI 结构体管理图形界面状态和组件
type UI struct {
	theme             *material.Theme
	idInput           widget.Editor
	queryBtn          widget.Clickable
	createBtn         widget.Clickable
	titleInput        widget.Editor
	contentInput      widget.Editor
	saveBtn           widget.Clickable
	deleteBtn         widget.Clickable
	db                *sql.DB
	errorMsg          string
	successMsg        string
	isCreating        bool
	articles          []Article
	selectedArticleID int
	sidebarCollapsed  bool
	toggleSidebarBtn  widget.Clickable
	newArticleBtn     widget.Clickable
	previewMode       bool
	togglePreviewBtn  widget.Clickable
	renderedHTML      string
	articleClickables map[int]*widget.Clickable
	renameBtn         widget.Clickable
	showRenameDialog  bool
	renameTitleInput  widget.Editor
	confirmRenameBtn  widget.Clickable
	cancelRenameBtn   widget.Clickable
	searchInput       widget.Editor
	searchBtn         widget.Clickable
	clearSearchBtn    widget.Clickable
	isSearching       bool
	articleList       widget.List
}

// NewUI 创建一个新的UI实例
func NewUI(db *sql.DB) *UI {
	ui := &UI{
		theme:             material.NewTheme(),
		db:                db,
		selectedArticleID: -1,
		articleClickables: make(map[int]*widget.Clickable),
	}
	ui.idInput.SingleLine = true
	ui.idInput.Submit = true
	ui.titleInput.SingleLine = true
	ui.renameTitleInput.SingleLine = true
	ui.searchInput.SingleLine = true
	ui.loadArticles()
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
	if ui.toggleSidebarBtn.Clicked(gtx) {
		ui.sidebarCollapsed = !ui.sidebarCollapsed
	}

	if ui.newArticleBtn.Clicked(gtx) {
		ui.createArticle()
	}

	if ui.deleteBtn.Clicked(gtx) {
		ui.deleteArticle()
	}

	if ui.saveBtn.Clicked(gtx) {
		ui.saveArticle()
	}

	if ui.togglePreviewBtn.Clicked(gtx) {
		ui.previewMode = !ui.previewMode
		if ui.previewMode {
			ui.updatePreview()
		}
	}

	if ui.renameBtn.Clicked(gtx) {
		if ui.selectedArticleID != -1 {
			ui.showRenameDialog = true
			ui.renameTitleInput.SetText(ui.titleInput.Text())
		} else {
			ui.errorMsg = "请先选择要重命名的文章"
		}
	}

	if ui.confirmRenameBtn.Clicked(gtx) {
		ui.confirmRename()
	}

	if ui.cancelRenameBtn.Clicked(gtx) {
		ui.showRenameDialog = false
		ui.renameTitleInput.SetText("")
	}

	if ui.searchBtn.Clicked(gtx) {
		ui.searchArticles()
	}

	if ui.clearSearchBtn.Clicked(gtx) {
		ui.clearSearch()
	}

	// 检查文章列表项的点击
	for _, article := range ui.articles {
		click := ui.articleClickables[article.ID]
		if click.Clicked(gtx) {
			ui.selectArticle(article.ID)
		}
	}

	layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(ui.sidebarLayout()),
		layout.Flexed(1, ui.editorLayout()),
	)

	if ui.showRenameDialog {
		ui.renameDialogLayout(gtx)
	}
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

	ui.titleInput.SetText(article.Title)
	ui.contentInput.SetText(article.Content)
	ui.errorMsg = ""
	ui.successMsg = fmt.Sprintf("创建时间: %s | 修改时间: %s", article.CreatedAt, article.UpdatedAt)
}

// createArticle 进入新建模式，清空输入框
func (ui *UI) createArticle() {
	ui.idInput.SetText("")
	ui.titleInput.SetText("")
	ui.contentInput.SetText("")
	ui.errorMsg = ""
	ui.successMsg = ""
	ui.isCreating = true
	ui.selectedArticleID = -1
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
		id, err := CreateArticle(ui.db, title, content)
		if err != nil {
			ui.errorMsg = fmt.Sprintf("创建失败: %v", err)
			return
		}
		ui.idInput.SetText(strconv.FormatInt(id, 10))
		ui.selectedArticleID = int(id)
		ui.isCreating = false
		ui.errorMsg = ""
		ui.successMsg = "创建成功！"
		ui.loadArticles()
	} else {
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
		ui.errorMsg = ""
		ui.successMsg = "保存成功！"
	}
}

// updatePreview 更新Markdown预览
func (ui *UI) updatePreview() {
	markdown := ui.contentInput.Text()
	html, err := MarkdownToHTML(markdown)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("预览生成失败: %v", err)
		return
	}
	ui.renderedHTML = html
}

// selectArticle 选择文章并加载到编辑器
func (ui *UI) selectArticle(id int) {
	ui.selectedArticleID = id
	article, err := GetArticleByID(ui.db, id)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("加载文章失败: %v", err)
		return
	}
	ui.titleInput.SetText(article.Title)
	ui.contentInput.SetText(article.Content)
	ui.idInput.SetText(strconv.Itoa(id))
	ui.isCreating = false
	ui.errorMsg = ""
	ui.successMsg = fmt.Sprintf("创建时间: %s | 修改时间: %s", article.CreatedAt, article.UpdatedAt)
}

// deleteArticle 删除当前选中的文章
func (ui *UI) deleteArticle() {
	if ui.selectedArticleID == -1 {
		ui.errorMsg = "请先选择要删除的文章"
		return
	}

	err := DeleteArticleByID(ui.db, ui.selectedArticleID)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("删除失败: %v", err)
		return
	}

	ui.errorMsg = ""
	ui.successMsg = "删除成功！"
	ui.selectedArticleID = -1
	ui.titleInput.SetText("")
	ui.contentInput.SetText("")
	ui.idInput.SetText("")
	ui.loadArticles()
}

// loadArticles 从数据库加载所有文章
func (ui *UI) loadArticles() {
	articles, err := GetAllArticles(ui.db)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("加载文章列表失败: %v", err)
		return
	}
	ui.articles = articles
	// 清空并重建 clickable map，确保每个文章都有新的 clickable
	ui.articleClickables = make(map[int]*widget.Clickable)
	for _, article := range articles {
		ui.articleClickables[article.ID] = &widget.Clickable{}
	}
}

// confirmRename 确认重命名
func (ui *UI) confirmRename() {
	newTitle := ui.renameTitleInput.Text()
	if newTitle == "" {
		ui.errorMsg = "标题不能为空"
		return
	}

	err := RenameArticleByID(ui.db, ui.selectedArticleID, newTitle)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("重命名失败: %v", err)
		return
	}

	ui.titleInput.SetText(newTitle)
	ui.showRenameDialog = false
	ui.renameTitleInput.SetText("")
	ui.errorMsg = ""
	ui.successMsg = "重命名成功！"
	ui.loadArticles()
}

// searchArticles 搜索文章
func (ui *UI) searchArticles() {
	keyword := ui.searchInput.Text()
	if keyword == "" {
		ui.errorMsg = "请输入搜索关键词"
		return
	}

	articles, err := SearchArticles(ui.db, keyword)
	if err != nil {
		ui.errorMsg = fmt.Sprintf("搜索失败: %v", err)
		return
	}

	ui.articles = articles
	ui.isSearching = true
	ui.errorMsg = ""
	ui.successMsg = fmt.Sprintf("找到 %d 篇文章", len(articles))
}

// clearSearch 清除搜索
func (ui *UI) clearSearch() {
	ui.searchInput.SetText("")
	ui.isSearching = false
	ui.loadArticles()
	ui.errorMsg = ""
	ui.successMsg = "已清除搜索"
}

// sidebarLayout 渲染侧边栏布局
func (ui *UI) sidebarLayout() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		gtx.Constraints.Max.X = 350 // 设置侧边栏宽度
		if ui.sidebarCollapsed {
			// 折叠状态：显示一个小按钮切换回展开状态
			gtx.Constraints.Max.X = 60
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{
					Axis: layout.Vertical,
				}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(ui.theme, &ui.toggleSidebarBtn, "☰")
						return btn.Layout(gtx)
					}),
					layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						btn := material.Button(ui.theme, &ui.newArticleBtn, "+")
						return btn.Layout(gtx)
					}),
				)
			})
		}

		return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis:      layout.Horizontal,
						Alignment: layout.Middle,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							btn := material.Button(ui.theme, &ui.toggleSidebarBtn, "☰")
							return btn.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Width: unit.Dp(10)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(ui.theme, unit.Sp(18), "文章列表")
							lbl.Font.Weight = font.Bold
							return lbl.Layout(gtx)
						}),
					)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.searchLayout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.articleListLayoutContent(gtx)
				}),
				layout.Flexed(1, layout.Spacer{}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(ui.theme, &ui.newArticleBtn, "新建文章")
					return btn.Layout(gtx)
				}),
			)
		})
	}
}

// searchLayout 渲染搜索框
func (ui *UI) searchLayout(gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			editor := material.Editor(ui.theme, &ui.searchInput, "搜索文章...")
			editor.TextSize = unit.Sp(14)
			return editor.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Horizontal,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(ui.theme, &ui.searchBtn, "搜索")
					return btn.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Width: unit.Dp(5)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					btn := material.Button(ui.theme, &ui.clearSearchBtn, "清除")
					return btn.Layout(gtx)
				}),
			)
		}),
	)
}

// articleListLayoutContent 渲染文章列表内容
func (ui *UI) articleListLayoutContent(gtx layout.Context) layout.Dimensions {
	if len(ui.articles) == 0 {
		lbl := material.Label(ui.theme, unit.Sp(14), "暂无文章")
		return lbl.Layout(gtx)
	}

	// 配置列表
	ui.articleList.Axis = layout.Vertical

	return ui.articleList.Layout(gtx, len(ui.articles), func(gtx layout.Context, index int) layout.Dimensions {
		article := ui.articles[index]
		return ui.articleItemLayout(gtx, article)
	})
}

// articleItemLayout 渲染单个文章列表项
func (ui *UI) articleItemLayout(gtx layout.Context, article Article) layout.Dimensions {
	click := ui.articleClickables[article.ID]
	isSelected := ui.selectedArticleID == article.ID

	// 默认白色背景，选中时为蓝色
	bgColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	if isSelected {
		bgColor = ui.theme.Palette.ContrastBg
	}

	// 只取日期部分，去掉具体时间
	dateStr := article.CreatedAt
	if len(dateStr) >= 10 {
		dateStr = dateStr[:10]
	}

	// 使用 clickable 包装内容
	return click.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// 背景
		stack := clip.Rect{
			Min: image.Point{},
			Max: gtx.Constraints.Max,
		}.Push(gtx.Ops)
		defer stack.Pop()
		paint.FillShape(gtx.Ops, bgColor, clip.Rect{
			Min: image.Point{},
			Max: gtx.Constraints.Max,
		}.Op())

		// 内容
		return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(ui.theme, unit.Sp(14), article.Title)
					if isSelected {
						lbl.Color = ui.theme.Palette.ContrastFg
					}
					lbl.Font.Weight = font.Medium
					return lbl.Layout(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(2)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(ui.theme, unit.Sp(12), dateStr)
					if isSelected {
						lbl.Color = ui.theme.Palette.ContrastFg
					}
					return lbl.Layout(gtx)
				}),
			)
		})
	})
}

// editorLayout 渲染编辑区域布局
func (ui *UI) editorLayout() layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.toolbarLayoutContent(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.messageLayoutContent(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return ui.titleEditorLayoutContent(gtx)
				}),
				layout.Rigid(layout.Spacer{Height: unit.Dp(10)}.Layout),
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.contentEditorLayoutContent(gtx)
				}),
			)
		})
	}
}

// toolbarLayoutContent 渲染工具栏内容
func (ui *UI) toolbarLayoutContent(gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(18), "编辑器")
			lbl.Font.Weight = font.Bold
			return lbl.Layout(gtx)
		}),
		layout.Flexed(1, layout.Spacer{}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(ui.theme, &ui.togglePreviewBtn, "预览")
				return btn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(ui.theme, &ui.renameBtn, "重命名")
				return btn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(ui.theme, &ui.saveBtn, "保存")
				return btn.Layout(gtx)
			})
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(ui.theme, &ui.deleteBtn, "删除")
				return btn.Layout(gtx)
			})
		}),
	)
}

// messageLayoutContent 渲染消息内容
func (ui *UI) messageLayoutContent(gtx layout.Context) layout.Dimensions {
	if ui.errorMsg == "" && ui.successMsg == "" {
		return layout.Dimensions{}
	}

	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.errorMsg != "" {
				lbl := material.Label(ui.theme, unit.Sp(14), ui.errorMsg)
				lbl.Color = ui.theme.ContrastBg
				return lbl.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.successMsg != "" {
				lbl := material.Label(ui.theme, unit.Sp(14), ui.successMsg)
				return lbl.Layout(gtx)
			}
			return layout.Dimensions{}
		}),
	)
}

// titleEditorLayoutContent 渲染标题编辑器内容
func (ui *UI) titleEditorLayoutContent(gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(16), "标题")
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
}

// contentEditorLayoutContent 渲染内容编辑器内容
func (ui *UI) contentEditorLayoutContent(gtx layout.Context) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Vertical,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(16), "内容")
			return lbl.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Height: unit.Dp(5)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if ui.previewMode {
				return ui.previewLayout(gtx)
			}
			contentEditor := material.Editor(ui.theme, &ui.contentInput, "文章内容")
			contentEditor.TextSize = unit.Sp(16)
			return contentEditor.Layout(gtx)
		}),
	)
}

// previewLayout 渲染Markdown预览
func (ui *UI) previewLayout(gtx layout.Context) layout.Dimensions {
	if ui.contentInput.Text() == "" {
		lbl := material.Label(ui.theme, unit.Sp(14), "点击预览按钮查看Markdown渲染效果")
		return lbl.Layout(gtx)
	}

	blocks, err := ParseMarkdownBlock(ui.contentInput.Text())
	if err != nil {
		lbl := material.Label(ui.theme, unit.Sp(14), "解析失败")
		return lbl.Layout(gtx)
	}

	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return ui.renderMarkdownBlocks(blocks)(gtx)
	})
}

// renderMarkdownBlocks 渲染Markdown块
func (ui *UI) renderMarkdownBlocks(blocks []MarkdownBlock) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for _, block := range blocks {
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return ui.renderMarkdownBlock(gtx, block)
			}))
		}
		return layout.Flex{
			Axis: layout.Vertical,
		}.Layout(gtx, children...)
	}
}

// renderMarkdownBlock 渲染单个Markdown块
func (ui *UI) renderMarkdownBlock(gtx layout.Context, block MarkdownBlock) layout.Dimensions {
	switch block.Type {
	case BlockTypeHeading:
		return ui.renderHeading(gtx, block)
	case BlockTypeParagraph:
		return ui.renderParagraph(gtx, block.Content)
	case BlockTypeCode:
		return ui.renderCodeBlock(gtx, block.Content)
	case BlockTypeList:
		return ui.renderListItem(gtx, block.Content)
	case BlockTypeQuote:
		return ui.renderQuote(gtx, block.Content)
	case BlockTypeHorizontalRule:
		return ui.renderHorizontalRule(gtx)
	default:
		return layout.Dimensions{}
	}
}

// renderHeading 渲染标题
func (ui *UI) renderHeading(gtx layout.Context, block MarkdownBlock) layout.Dimensions {
	fontSize := unit.Sp(24 - block.Level*2)
	fontWeight := font.Bold

	lbl := material.Label(ui.theme, fontSize, block.Content)
	lbl.Font.Weight = fontWeight
	return lbl.Layout(gtx)
}

// renderParagraph 渲染段落
func (ui *UI) renderParagraph(gtx layout.Context, text string) layout.Dimensions {
	lbl := material.Label(ui.theme, unit.Sp(16), text)
	return lbl.Layout(gtx)
}

// renderCodeBlock 渲染代码块
func (ui *UI) renderCodeBlock(gtx layout.Context, code string) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		stack := clip.Rect{
			Min: image.Point{},
			Max: gtx.Constraints.Max,
		}.Push(gtx.Ops)
		defer stack.Pop()
		paint.FillShape(gtx.Ops, ui.theme.Palette.ContrastBg, clip.Rect{
			Min: image.Point{},
			Max: gtx.Constraints.Max,
		}.Op())

		lbl := material.Label(ui.theme, unit.Sp(14), code)
		lbl.Font.Typeface = "monospace"
		return lbl.Layout(gtx)
	})
}

// renderListItem 渲染列表项
func (ui *UI) renderListItem(gtx layout.Context, text string) layout.Dimensions {
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(16), "• ")
			return lbl.Layout(gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(16), text)
			return lbl.Layout(gtx)
		}),
	)
}

// renderQuote 渲染引用
func (ui *UI) renderQuote(gtx layout.Context, text string) layout.Dimensions {
	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		stack := clip.Rect{
			Min: image.Point{},
			Max: gtx.Constraints.Max,
		}.Push(gtx.Ops)
		defer stack.Pop()
		paint.FillShape(gtx.Ops, ui.theme.Palette.ContrastBg, clip.Rect{
			Min: image.Point{},
			Max: gtx.Constraints.Max,
		}.Op())

		lbl := material.Label(ui.theme, unit.Sp(16), text)
		lbl.Font.Style = font.Italic
		return lbl.Layout(gtx)
	})
}

// renderHorizontalRule 渲染水平线
func (ui *UI) renderHorizontalRule(gtx layout.Context) layout.Dimensions {
	return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				stack := clip.Rect{
					Min: image.Point{},
					Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Max.Y},
				}.Push(gtx.Ops)
				defer stack.Pop()
				paint.FillShape(gtx.Ops, ui.theme.Palette.ContrastBg, clip.Rect{
					Min: image.Point{},
					Max: image.Point{X: gtx.Constraints.Max.X, Y: gtx.Constraints.Max.Y},
				}.Op())
				return layout.Dimensions{Size: gtx.Constraints.Max}
			}),
		)
	})
}

// renameDialogLayout 渲染重命名对话框
func (ui *UI) renameDialogLayout(gtx layout.Context) layout.Dimensions {
	return layout.Stack{}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			stack := clip.Rect{
				Min: image.Point{},
				Max: gtx.Constraints.Max,
			}.Push(gtx.Ops)
			defer stack.Pop()
			paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 0, B: 0, A: 200}, clip.Rect{
				Min: image.Point{},
				Max: gtx.Constraints.Max,
			}.Op())
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return layout.UniformInset(unit.Dp(20)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{
						Axis: layout.Vertical,
					}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lbl := material.Label(ui.theme, unit.Sp(18), "重命名文章")
							lbl.Font.Weight = font.Bold
							return lbl.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							editor := material.Editor(ui.theme, &ui.renameTitleInput, "新标题")
							editor.TextSize = unit.Sp(16)
							return editor.Layout(gtx)
						}),
						layout.Rigid(layout.Spacer{Height: unit.Dp(15)}.Layout),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{
								Axis: layout.Horizontal,
							}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										btn := material.Button(ui.theme, &ui.confirmRenameBtn, "确认")
										return btn.Layout(gtx)
									})
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
										btn := material.Button(ui.theme, &ui.cancelRenameBtn, "取消")
										return btn.Layout(gtx)
									})
								}),
							)
						}),
					)
				})
			})
		}),
	)
}
