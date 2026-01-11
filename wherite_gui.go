package main

import (
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"strconv"
	"strings"
	"time"

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
	successMsgTime    time.Time
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
	searchInput       widget.Editor
	searchBtn         widget.Clickable
	clearSearchBtn    widget.Clickable
	isSearching       bool
	articleList       widget.List
	previewList       widget.List
	hasUnsavedChanges bool   // 是否有未保存的更改
	originalTitle     string // 文章原始标题（用于比较）
	originalContent   string // 文章原始内容（用于比较）
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
	ui.titleInput.SingleLine = false
	ui.titleInput.Submit = true
	ui.searchInput.SingleLine = true
	ui.loadArticles()
	return ui
}

// Run 运行图形界面事件循环
func (ui *UI) Run(w *app.Window, db *sql.DB) error {
	var ops op.Ops

	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			// 窗口销毁时关闭数据库连接
			if db != nil {
				db.Close()
			}
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
	// 自动隐藏成功消息（1秒后）
	if !ui.successMsgTime.IsZero() && time.Since(ui.successMsgTime) >= time.Second {
		ui.successMsg = ""
		ui.successMsgTime = time.Time{}
	}

	// 检查是否有未保存的更改
	ui.hasUnsavedChanges = (ui.titleInput.Text() != ui.originalTitle) || (ui.contentInput.Text() != ui.originalContent)

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
		ui.successMsgTime = time.Now()
		ui.originalTitle = title
		ui.originalContent = content
		ui.hasUnsavedChanges = false
		ui.loadArticles()

		// 重新获取文章数据以显示正确的时间信息
		if article, err := GetArticleByID(ui.db, int(id)); err == nil {
			ui.successMsg = fmt.Sprintf("创建成功！ | 创建时间: %s | 修改时间: %s", article.CreatedAt, article.UpdatedAt)
		}
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
		ui.successMsgTime = time.Now()
		ui.originalTitle = title
		ui.originalContent = content
		ui.hasUnsavedChanges = false
		ui.loadArticles()

		// 重新获取文章数据以显示正确的时间信息
		if article, err := GetArticleByID(ui.db, id); err == nil {
			ui.successMsg = fmt.Sprintf("保存成功！ | 创建时间: %s | 修改时间: %s", article.CreatedAt, article.UpdatedAt)
		}
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
	ui.originalTitle = article.Title
	ui.originalContent = article.Content
	ui.hasUnsavedChanges = false
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
		// 设置侧边栏固定宽度为400
		gtx.Constraints.Min.X = 400
		gtx.Constraints.Max.X = 400
		if ui.sidebarCollapsed {
			// 折叠状态：显示一个小按钮切换回展开状态
			gtx.Constraints.Min.X = 60
			gtx.Constraints.Max.X = 60
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(ui.theme, &ui.toggleSidebarBtn, "☰")
				return btn.Layout(gtx)
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
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return ui.articleListLayoutContent(gtx)
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
					// 标题最多3行，每行约16个中文字符宽度
					lbl.MaxLines = 3
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
			// 如果没有选择文章且不在创建模式，显示欢迎页面覆盖整个编辑器区域
			if ui.selectedArticleID == -1 && !ui.isCreating {
				return ui.editorWelcomeLayout(gtx)
			}

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

// editorWelcomeLayout 渲染编辑器欢迎页面（覆盖整个编辑器区域）
func (ui *UI) editorWelcomeLayout(gtx layout.Context) layout.Dimensions {
	// 绘制背景色
	stack := clip.Rect{
		Min: image.Point{},
		Max: gtx.Constraints.Max,
	}.Push(gtx.Ops)
	paint.FillShape(gtx.Ops, color.NRGBA{R: 250, G: 250, B: 250, A: 255}, clip.Rect{
		Min: image.Point{},
		Max: gtx.Constraints.Max,
	}.Op())
	stack.Pop()

	// 测量内容高度
	content := func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(ui.theme, unit.Sp(24), "欢迎使用 Wherite")
				lbl.Font.Weight = font.Bold
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(ui.theme, unit.Sp(14), "选择一个文章开始编辑，或创建新文章")
				lbl.Color = color.NRGBA{R: 136, G: 136, B: 136, A: 255}
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(32)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				btn := material.Button(ui.theme, &ui.newArticleBtn, "新建文章")
				btn.TextSize = unit.Sp(16)
				return btn.Layout(gtx)
			}),
		)
	}

	// 测量内容尺寸
	macro := op.Record(gtx.Ops)
	contentDims := content(gtx)
	macro.Stop()

	// 计算垂直居中的间距
	availHeight := gtx.Constraints.Max.Y
	topSpacing := (availHeight - contentDims.Size.Y) / 2
	if topSpacing < 0 {
		topSpacing = 0
	}

	// 使用 Stack 实现居中
	return layout.Stack{Alignment: layout.Center}.Layout(gtx,
		layout.Expanded(func(gtx layout.Context) layout.Dimensions {
			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			// 垂直方向：顶部间距
			gtx.Constraints.Min.Y = topSpacing
			layout.Spacer{Height: unit.Dp(float32(topSpacing))}.Layout(gtx)

			// 水平方向居中 + 内容
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Flexed(1, layout.Spacer{}.Layout),
				layout.Rigid(content),
				layout.Flexed(1, layout.Spacer{}.Layout),
			)
		}),
	)
}

// toolbarLayoutContent 渲染工具栏内容
func (ui *UI) toolbarLayoutContent(gtx layout.Context) layout.Dimensions {
	title := "编辑器"
	if ui.previewMode {
		title = "编辑器 -- 预览模式"
	}

	return layout.Flex{
		Axis:      layout.Horizontal,
		Alignment: layout.Middle,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(18), title)
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
				btn := material.Button(ui.theme, &ui.newArticleBtn, "新建")
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
	if ui.errorMsg == "" && ui.successMsg == "" && !ui.hasUnsavedChanges {
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
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if ui.hasUnsavedChanges {
				lbl := material.Label(ui.theme, unit.Sp(14), "未保存……")
				lbl.Color = color.NRGBA{R: 255, G: 165, B: 0, A: 255} // 橙色提示
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
			// 标签背景色 - 使用更明显的灰色
			bgColor := color.NRGBA{R: 220, G: 220, B: 220, A: 255}
			height := gtx.Dp(30)
			width := gtx.Constraints.Max.X

			// 直接绘制背景
			stack := clip.Rect{
				Min: image.Point{},
				Max: image.Point{X: width, Y: height},
			}.Push(gtx.Ops)
			paint.FillShape(gtx.Ops, bgColor, clip.Rect{
				Min: image.Point{},
				Max: image.Point{X: width, Y: height},
			}.Op())
			stack.Pop()

			lbl := material.Label(ui.theme, unit.Sp(16), "标题")
			lbl.Color = color.NRGBA{R: 60, G: 60, B: 60, A: 255}
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, lbl.Layout)
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
			// 标签背景色 - 使用更明显的灰色
			bgColor := color.NRGBA{R: 220, G: 220, B: 220, A: 255}
			height := gtx.Dp(30)
			width := gtx.Constraints.Max.X

			// 直接绘制背景
			stack := clip.Rect{
				Min: image.Point{},
				Max: image.Point{X: width, Y: height},
			}.Push(gtx.Ops)
			paint.FillShape(gtx.Ops, bgColor, clip.Rect{
				Min: image.Point{},
				Max: image.Point{X: width, Y: height},
			}.Op())
			stack.Pop()

			lbl := material.Label(ui.theme, unit.Sp(16), "内容")
			lbl.Color = color.NRGBA{R: 60, G: 60, B: 60, A: 255}
			return layout.UniformInset(unit.Dp(5)).Layout(gtx, lbl.Layout)
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

	if len(blocks) == 0 {
		lbl := material.Label(ui.theme, unit.Sp(14), "无内容")
		return lbl.Layout(gtx)
	}

	// 配置预览列表为可滚动
	ui.previewList.Axis = layout.Vertical

	return ui.previewList.Layout(gtx, len(blocks), func(gtx layout.Context, index int) layout.Dimensions {
		return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return ui.renderMarkdownBlock(gtx, blocks[index])
		})
	})
}

// renderMarkdownBlocks 渲染Markdown块
func (ui *UI) renderMarkdownBlocks(blocks []MarkdownBlock) layout.Widget {
	return func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild
		for i, block := range blocks {
			if i > 0 {
				// 代码块之间添加更大间距
				spacing := unit.Dp(8)
				if block.Type == BlockTypeCode {
					spacing = unit.Dp(16)
				}
				children = append(children, layout.Rigid(layout.Spacer{Height: spacing}.Layout))
			}
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
		return ui.renderCodeBlock(gtx, block)
	case BlockTypeList:
		return ui.renderListItem(gtx, block.Content)
	case BlockTypeTaskList:
		return ui.renderTaskItem(gtx, block)
	case BlockTypeQuote:
		return ui.renderQuote(gtx, block.Content)
	case BlockTypeHorizontalRule:
		return ui.renderHorizontalRule(gtx)
	case BlockTypeTable:
		return ui.renderTable(gtx, block)
	default:
		return layout.Dimensions{}
	}
}

// renderHeading 渲染标题
func (ui *UI) renderHeading(gtx layout.Context, block MarkdownBlock) layout.Dimensions {
	fontSize := unit.Sp(24 - block.Level*2)
	fontWeight := font.Bold

	if len(block.Inlines) > 0 {
		return ui.renderInlines(gtx, block.Inlines, fontSize)
	}

	lbl := material.Label(ui.theme, fontSize, block.Content)
	lbl.Font.Weight = fontWeight
	return lbl.Layout(gtx)
}

// renderParagraph 渲染段落
func (ui *UI) renderParagraph(gtx layout.Context, text string) layout.Dimensions {
	if len(text) == 0 {
		return layout.Dimensions{}
	}

	// 解析行内元素
	inlines := ParseInlines(text)
	if len(inlines) > 0 {
		return ui.renderInlines(gtx, inlines, unit.Sp(16))
	}

	lbl := material.Label(ui.theme, unit.Sp(16), text)
	return lbl.Layout(gtx)
}

// renderInlines 渲染行内元素
func (ui *UI) renderInlines(gtx layout.Context, inlines []InlineElement, baseSize unit.Sp) layout.Dimensions {
	var children []layout.FlexChild
	for _, inline := range inlines {
		inlineCopy := inline
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return ui.renderInline(gtx, inlineCopy, baseSize)
		}))
	}
	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx, children...)
}

// renderInline 渲染单个行内元素
func (ui *UI) renderInline(gtx layout.Context, inline InlineElement, baseSize unit.Sp) layout.Dimensions {
	size := baseSize
	text := inline.Text

	// 检查是否是加粗斜体组合
	isBoldItalic := int(inline.Type)&0x10 != 0
	isItalic := inline.Type == InlineTypeItalic || isBoldItalic

	switch inline.Type {
	case InlineTypeBold, InlineTypeBold | 0x10:
		lbl := material.Label(ui.theme, size, text)
		lbl.Font.Weight = font.Bold
		// 绘制背景（在文字之前）
		bgColor := color.NRGBA{}
		ui.drawInlineBackground(gtx, lbl, bgColor)
		dims := lbl.Layout(gtx)
		// 斜体使用下划线表示
		if isItalic {
			ui.drawItalicTransform(gtx, dims)
		}
		return dims
	case InlineTypeItalic:
		lbl := material.Label(ui.theme, size, text)
		// 绘制背景
		ui.drawInlineBackground(gtx, lbl, color.NRGBA{})
		dims := lbl.Layout(gtx)
		// 斜体使用下划线表示
		ui.drawItalicTransform(gtx, dims)
		return dims
	case InlineTypeStrike:
		lbl := material.Label(ui.theme, size, text)
		// 绘制删除线
		dims := lbl.Layout(gtx)
		textSize := dims.Size
		if textSize.X > 0 && textSize.Y > 0 {
			lineOp := clip.Rect{
				Min: image.Point{X: 0, Y: textSize.Y/2 - 1},
				Max: image.Point{X: textSize.X, Y: textSize.Y/2 + 1},
			}.Op()
			paint.FillShape(gtx.Ops, color.NRGBA{R: 128, G: 128, B: 128, A: 255}, lineOp)
		}
		return dims
	case InlineTypeCode:
		lbl := material.Label(ui.theme, size, text)
		lbl.Font.Typeface = "monospace"
		// 先绘制背景
		ui.drawInlineBackground(gtx, lbl, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
		// 再绘制文字
		return lbl.Layout(gtx)
	case InlineTypeLink:
		// 渲染为 "text (url)" 格式
		displayText := text
		if inline.URL != "" {
			displayText = text + " (" + inline.URL + ")"
		}
		lbl := material.Label(ui.theme, size, displayText)
		lbl.Color = color.NRGBA{R: 0, G: 122, B: 255, A: 255} // 蓝色链接颜色
		return lbl.Layout(gtx)
	default:
		lbl := material.Label(ui.theme, size, text)
		return lbl.Layout(gtx)
	}
}

// drawItalicTransform 绘制斜体变换效果
// 注意：Gio 的 material 主题不直接支持斜体，使用下划线作为视觉提示
func (ui *UI) drawItalicTransform(gtx layout.Context, dims layout.Dimensions) {
	textSize := dims.Size
	if textSize.X > 0 && textSize.Y > 0 {
		// 绘制下划线表示斜体
		underlineY := textSize.Y - 4
		if underlineY < textSize.Y-2 {
			underlineY = textSize.Y - 2
		}
		lineOp := clip.Rect{
			Min: image.Point{X: 0, Y: underlineY},
			Max: image.Point{X: textSize.X, Y: underlineY + 1},
		}.Op()
		paint.FillShape(gtx.Ops, color.NRGBA{R: 100, G: 100, B: 200, A: 255}, lineOp)
	}
}

// drawInlineBackground 绘制行内元素背景（先于文字绘制）
func (ui *UI) drawInlineBackground(gtx layout.Context, lbl material.LabelStyle, bgColor color.NRGBA) {
	if bgColor.A == 0 {
		return
	}
	// 使用记录操作来获取文字尺寸
	macro := op.Record(gtx.Ops)
	dims := lbl.Layout(gtx)
	macro.Stop()

	textSize := dims.Size
	if textSize.X > 0 && textSize.Y > 0 {
		stack := clip.Rect{
			Min: image.Point{X: 0, Y: 2},
			Max: image.Point{X: textSize.X, Y: textSize.Y - 2},
		}.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, bgColor, clip.Rect{
			Min: image.Point{X: 0, Y: 2},
			Max: image.Point{X: textSize.X, Y: textSize.Y - 2},
		}.Op())
		stack.Pop()
	}
}

// renderCodeBlock 渲染代码块
func (ui *UI) renderCodeBlock(gtx layout.Context, block MarkdownBlock) layout.Dimensions {
	code := block.Content

	// 分割代码为多行
	lines := splitLines(code)
	if len(lines) == 0 {
		return layout.Dimensions{}
	}

	// 提取语言标识符（第一行可能是语言名，如 ```go）
	codeLines := lines
	if len(lines) > 0 && len(lines[0]) > 3 && lines[0][:3] == "```" {
		// 第一行是代码块标记 ```lang，提取语言名并跳过这行
		lang := strings.TrimSpace(lines[0][3:])
		if len(lang) > 0 {
			// 有语言标识符，跳过第一行
			codeLines = lines[1:]
		}
	}
	// 过滤掉可能的结束标记行（```）
	for len(codeLines) > 0 && len(codeLines[0]) >= 3 && codeLines[0][:3] == "```" {
		codeLines = codeLines[1:]
	}

	// 渲染背景
	bgColor := color.NRGBA{R: 245, G: 245, B: 245, A: 255} // 浅灰色背景
	codePadding := unit.Dp(8)                              // 代码内容内边距

	return layout.UniformInset(unit.Dp(12)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// 先渲染代码内容（带内边距）获取尺寸
		var codeDims layout.Dimensions
		var codeWidget layout.Widget = func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: codePadding, Right: codePadding, Top: codePadding, Bottom: codePadding}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				var children []layout.FlexChild
				for i, line := range codeLines {
					if i > 0 {
						children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
					}
					lineCopy := line
					children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						lbl := material.Label(ui.theme, unit.Sp(14), lineCopy)
						lbl.Font.Typeface = "monospace"
						return lbl.Layout(gtx)
					}))
				}
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
			})
		}
		codeDims = codeWidget(gtx)

		// 绘制背景（使用内容尺寸）
		paint.FillShape(gtx.Ops, bgColor, clip.Rect{
			Min: image.Point{},
			Max: image.Point{X: codeDims.Size.X, Y: codeDims.Size.Y},
		}.Op())

		// 重新渲染代码内容（现在背景已绘制）
		return codeWidget(gtx)
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

// renderTaskItem 渲染任务列表项
func (ui *UI) renderTaskItem(gtx layout.Context, block MarkdownBlock) layout.Dimensions {
	if block.TaskData == nil {
		return layout.Dimensions{}
	}

	taskData := block.TaskData
	checkbox := "☐"
	if taskData.Checked {
		checkbox = "☑"
	}

	return layout.Flex{
		Axis: layout.Horizontal,
	}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(16), checkbox)
			if taskData.Checked {
				lbl.Color = color.NRGBA{R: 0, G: 150, B: 0, A: 255}
			}
			return lbl.Layout(gtx)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			lbl := material.Label(ui.theme, unit.Sp(16), taskData.Content)
			if taskData.Checked {
				// 已完成的任务使用灰色删除线效果
				lbl.Color = color.NRGBA{R: 150, G: 150, B: 150, A: 255}
			}
			return lbl.Layout(gtx)
		}),
	)
}

// renderQuote 渲染引用
func (ui *UI) renderQuote(gtx layout.Context, text string) layout.Dimensions {
	// 分割多行
	lines := splitLines(text)
	if len(lines) == 0 {
		return layout.Dimensions{}
	}

	return layout.UniformInset(unit.Dp(10)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		var children []layout.FlexChild

		for lineIndex, line := range lines {
			if lineIndex > 0 {
				children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))
			}

			// 渲染每一行
			children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lineCopy := line
				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					// 左边框
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						// 绘制左边框，只在文字高度范围内
						dims := material.Label(ui.theme, unit.Sp(16), " ").Layout(gtx)
						borderWidth := 4
						borderHeight := dims.Size.Y
						if borderHeight == 0 {
							borderHeight = 16
						}
						stack := clip.Rect{
							Min: image.Point{},
							Max: image.Point{X: borderWidth, Y: borderHeight},
						}.Push(gtx.Ops)
						paint.FillShape(gtx.Ops, ui.theme.Palette.ContrastBg, clip.Rect{
							Min: image.Point{},
							Max: image.Point{X: borderWidth, Y: borderHeight},
						}.Op())
						stack.Pop()
						return layout.Dimensions{Size: image.Point{X: borderWidth, Y: borderHeight}}
					}),
					// 间距
					layout.Rigid(layout.Spacer{Width: unit.Dp(8)}.Layout),
					// 文字内容
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						if len(lineCopy) > 0 {
							inlines := ParseInlines(lineCopy)
							if len(inlines) > 0 {
								return ui.renderInlines(gtx, inlines, unit.Sp(16))
							}
						}
						lbl := material.Label(ui.theme, unit.Sp(16), lineCopy)
						return lbl.Layout(gtx)
					}),
				)
			}))
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
	})
}

// renderHorizontalRule 渲染水平线
func (ui *UI) renderHorizontalRule(gtx layout.Context) layout.Dimensions {
	height := 2
	return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(4), Bottom: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			// 使用可用的最大宽度
			width := gtx.Constraints.Max.X
			if width > 600 {
				width = 600
			}
			paint.FillShape(gtx.Ops, color.NRGBA{R: 200, G: 200, B: 200, A: 255}, clip.Rect{
				Min: image.Point{},
				Max: image.Point{X: width, Y: height},
			}.Op())
			return layout.Dimensions{Size: image.Point{X: width, Y: height}}
		})
	})
}

// renderTable 渲染表格
func (ui *UI) renderTable(gtx layout.Context, block MarkdownBlock) layout.Dimensions {
	if block.TableData == nil || len(block.TableData.Headers) == 0 {
		return layout.Dimensions{}
	}

	tableData := block.TableData
	headers := tableData.Headers
	rows := tableData.Rows

	// 计算列数（取最大列数）
	colCount := len(headers)
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}

	// 估算每列宽度
	colWidth := gtx.Constraints.Max.X / colCount
	if colWidth > 200 {
		colWidth = 200
	}

	borderColor := color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	headerBgColor := color.NRGBA{R: 240, G: 240, B: 240, A: 255}
	cellPadding := unit.Dp(6)
	cellPaddingPx := int(gtx.Dp(cellPadding))
	vertPadding := int(gtx.Dp(4))

	// 渲染表头
	var tableChildren []layout.FlexChild

	// 表头行
	tableChildren = append(tableChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		headerCells := make([]layout.FlexChild, len(headers))
		for j, header := range headers {
			headerCopy := header
			headerCells[j] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(ui.theme, unit.Sp(14), headerCopy)
				lbl.Font.Weight = font.Bold
				return ui.renderTableCell(gtx, lbl, headerBgColor, borderColor, cellPaddingPx, vertPadding)
			})
		}
		return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, headerCells...)
	}))

	// 数据行
	for _, row := range rows {
		rowCopy := row
		tableChildren = append(tableChildren, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			rowCells := make([]layout.FlexChild, colCount)
			for j := 0; j < colCount; j++ {
				cellText := ""
				if j < len(rowCopy) {
					cellText = rowCopy[j]
				}
				cellCopy := cellText
				rowCells[j] = layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					lbl := material.Label(ui.theme, unit.Sp(14), cellCopy)
					return ui.renderTableCell(gtx, lbl, color.NRGBA{}, borderColor, cellPaddingPx, vertPadding)
				})
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, rowCells...)
		}))
	}

	return layout.UniformInset(unit.Dp(5)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx, tableChildren...)
	})
}

// renderTableCell 渲染单个表格单元格（带边框）
func (ui *UI) renderTableCell(gtx layout.Context, lbl material.LabelStyle, bgColor color.NRGBA, borderColor color.NRGBA, paddingPx int, vertPadding int) layout.Dimensions {
	return layout.Inset{Left: unit.Dp(float32(paddingPx)), Right: unit.Dp(float32(paddingPx)), Top: unit.Dp(float32(vertPadding)), Bottom: unit.Dp(float32(vertPadding))}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		// 获取文字尺寸
		macro := op.Record(gtx.Ops)
		dims := lbl.Layout(gtx)
		macro.Stop()

		// 绘制背景
		if bgColor.A > 0 {
			bgRect := clip.Rect{
				Min: image.Point{X: -paddingPx + 1, Y: -vertPadding + 1},
				Max: image.Point{X: dims.Size.X + paddingPx - 1, Y: dims.Size.Y + vertPadding - 1},
			}
			stack := bgRect.Push(gtx.Ops)
			paint.FillShape(gtx.Ops, bgColor, bgRect.Op())
			stack.Pop()
		}

		// 绘制边框（左边框、上边框、右边框、下边框）
		borderWidth := 1
		innerWidth := dims.Size.X + paddingPx*2 - 2
		innerHeight := dims.Size.Y + vertPadding*2 - 2

		// 上边框
		if vertPadding > borderWidth {
			topLine := clip.Rect{
				Min: image.Point{X: -paddingPx + 1, Y: -vertPadding + 1},
				Max: image.Point{X: -paddingPx + 1 + innerWidth, Y: -vertPadding + 1 + borderWidth},
			}
			stack := topLine.Push(gtx.Ops)
			paint.FillShape(gtx.Ops, borderColor, topLine.Op())
			stack.Pop()
		}

		// 左边框
		leftLine := clip.Rect{
			Min: image.Point{X: -paddingPx + 1, Y: -vertPadding + 1},
			Max: image.Point{X: -paddingPx + 1 + borderWidth, Y: -vertPadding + 1 + innerHeight},
		}
		stack := leftLine.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, borderColor, leftLine.Op())
		stack.Pop()

		// 右边框
		rightLine := clip.Rect{
			Min: image.Point{X: -paddingPx + 1 + innerWidth - borderWidth, Y: -vertPadding + 1},
			Max: image.Point{X: -paddingPx + 1 + innerWidth, Y: -vertPadding + 1 + innerHeight},
		}
		stack = rightLine.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, borderColor, rightLine.Op())
		stack.Pop()

		// 下边框
		bottomLine := clip.Rect{
			Min: image.Point{X: -paddingPx + 1, Y: -vertPadding + 1 + innerHeight - borderWidth},
			Max: image.Point{X: -paddingPx + 1 + innerWidth, Y: -vertPadding + 1 + innerHeight},
		}
		stack = bottomLine.Push(gtx.Ops)
		paint.FillShape(gtx.Ops, borderColor, bottomLine.Op())
		stack.Pop()

		// 渲染文字
		return lbl.Layout(gtx)
	})
}
