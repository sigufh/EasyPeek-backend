// api/news.go
package api

import (
	"strconv" // 用于字符串和数字转换

	"github.com/EasyPeek/EasyPeek-backend/internal/models"   // 导入新闻模型和请求/响应结构体
	"github.com/EasyPeek/EasyPeek-backend/internal/services" // 导入新闻服务
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"    // 导入公共工具函数，用于标准化的API响应
	"github.com/gin-gonic/gin"                               // 导入 Gin 框架
)

// NewsHandler 结构体，用于封装与新闻相关的 HTTP 请求处理逻辑
type NewsHandler struct {
	newsService *services.NewsService // 依赖 NewsService 来处理业务逻辑
}

// NewNewsHandler 创建并返回一个新的 NewsHandler 实例
func NewNewsHandler() *NewsHandler {
	return &NewsHandler{
		newsService: services.NewNewsService(), // 初始化 NewsService
	}
}

func (h *NewsHandler) CreateNews(c *gin.Context) {
	var req models.NewsCreateRequest
	// 将请求的 JSON 主体绑定到 NewsCreateRequest 结构体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	// 从 Gin 上下文中获取用户ID，假设认证中间件已将用户ID存储在其中
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not authenticated") // 如果用户未认证，返回未认证错误
		return
	}
	// 将 userID 转换为 uint 类型
	creatorID, ok := userID.(uint)
	if !ok {
		utils.InternalServerError(c, "Failed to get user ID from context")
		return
	}

	// 调用 NewsService 的 CreateNews 方法来创建新闻
	news, err := h.newsService.CreateNews(&req, creatorID)
	if err != nil {
		// 根据错误类型返回不同的 HTTP 状态码
		if err.Error() == "database connection not initialized" {
			utils.InternalServerError(c, err.Error())
		} else {
			utils.BadRequest(c, err.Error()) // 通常是业务逻辑错误，如数据重复
		}
		return
	}

	// 成功创建，返回新闻的响应格式
	utils.Success(c, news.ToResponse()) // 返回 201 Created 状态码
}

func (h *NewsHandler) GetNewsByID(c *gin.Context) {
	// 从 URL 参数中获取新闻ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr) // 将字符串ID转换为整数
	if err != nil {
		utils.BadRequest(c, "Invalid news ID") // 如果ID无效，返回错误
		return
	}

	// 调用 NewsService 的 GetNewsByID 方法
	news, err := h.newsService.GetNewsByID(uint(id))
	if err != nil {
		if err.Error() == "news not found" {
			utils.NotFound(c, err.Error()) // 如果新闻未找到，返回 404
		} else {
			utils.InternalServerError(c, err.Error()) // 其他数据库错误，返回 500
		}
		return
	}

	// 成功获取，返回新闻的响应格式
	utils.Success(c, news.ToResponse())
}

func (h *NewsHandler) GetAllNews(c *gin.Context) {
	// 获取查询参数中的页码和每页大小，并设置默认值
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	// 转换页码和每页大小为整数，并处理无效值
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 { // 限制每页最大大小，防止过大查询
		size = 10
	}

	// 调用 NewsService 的 GetAllNews 方法获取新闻列表和总数
	newsList, total, err := h.newsService.GetAllNews(page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error()) // 数据库或其他内部错误
		return
	}

	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	// 返回带分页信息成功的响应
	utils.SuccessWithPagination(c, newsResponses, total, page, size)
}

func (h *NewsHandler) UpdateNews(c *gin.Context) {
	// 从 URL 参数中获取新闻ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid news ID")
		return
	}

	var req models.NewsUpdateRequest
	// 将请求的 JSON 主体绑定到 NewsUpdateRequest 结构体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	// 先尝试获取要更新的新闻记录
	news, err := h.newsService.GetNewsByID(uint(id))
	if err != nil {
		if err.Error() == "news not found" {
			utils.NotFound(c, err.Error())
		} else {
			utils.InternalServerError(c, err.Error())
		}
		return
	}

	// 调用 NewsService 的 UpdateNews 方法进行更新
	// UpdateNews 接收的是现有新闻对象和更新请求
	if err := h.newsService.UpdateNews(news, &req); err != nil {
		utils.InternalServerError(c, err.Error()) // 更新失败通常是数据库错误
		return
	}

	// 成功更新，返回更新后的新闻响应格式
	utils.Success(c, news.ToResponse())
}

func (h *NewsHandler) DeleteNews(c *gin.Context) {
	// 从 URL 参数中获取新闻ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		utils.BadRequest(c, "Invalid news ID")
		return
	}

	// 调用 NewsService 的 DeleteNews 方法进行软删除
	if err := h.newsService.DeleteNews(uint(id)); err != nil {
		if err.Error() == "news not found or already deleted" {
			utils.NotFound(c, err.Error()) // 如果记录不存在或已删除，返回 404
		} else {
			utils.InternalServerError(c, err.Error()) // 其他数据库错误
		}
		return
	}

	// 成功删除，返回成功消息
	utils.Success(c, gin.H{"message": "News deleted successfully"})
}

func (h *NewsHandler) SearchNews(c *gin.Context) {
	// 获取查询参数中的搜索关键词
	queryStr := c.Query("query")
	if queryStr == "" {
		utils.BadRequest(c, "Search query cannot be empty")
		return
	}

	// 获取查询参数中的页码和每页大小，并设置默认值
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	// 转换页码和每页大小为整数，并处理无效值
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	// 调用 NewsService 的 SearchNews 方法进行搜索
	newsList, total, err := h.newsService.SearchNews(queryStr, page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error()) // 数据库或其他内部错误
		return
	}

	// 将搜索结果转换为响应格式
	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	// 返回带分页信息成功的响应
	utils.SuccessWithPagination(c, newsResponses, total, page, size)
}

// GetNewsByTitle 根据标题获取新闻
func (h *NewsHandler) GetNewsByTitle(c *gin.Context) {
	title := c.Query("title")
	if title == "" {
		utils.BadRequest(c, "Title parameter is required")
		return
	}

	newsList, err := h.newsService.GetNewsByTitle(title)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	utils.Success(c, newsResponses)
}

// GetNewsByCategory 根据分类获取新闻
func (h *NewsHandler) GetNewsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		utils.BadRequest(c, "Category is required")
		return
	}

	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	// 使用按分类查询的服务方法
	newsList, total, err := h.newsService.GetNewsByCategory(category, page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	utils.SuccessWithPagination(c, newsResponses, total, page, size)
}

// GetUnlinkedNews 获取未关联事件的新闻
func (h *NewsHandler) GetUnlinkedNews(c *gin.Context) {
	// 获取分页参数
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	newsList, total, err := h.newsService.GetUnlinkedNews(page, size)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	utils.SuccessWithPagination(c, newsResponses, total, page, size)
}

// UpdateNewsEventAssociation 批量更新新闻事件关联
func (h *NewsHandler) UpdateNewsEventAssociation(c *gin.Context) {
	var req struct {
		NewsIDs []uint `json:"news_ids" binding:"required"`
		EventID *uint  `json:"event_id"` // 可为空，表示取消关联
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request data: "+err.Error())
		return
	}

	if len(req.NewsIDs) == 0 {
		utils.BadRequest(c, "News IDs cannot be empty")
		return
	}

	err := h.newsService.UpdateNewsEventAssociationByIDs(req.NewsIDs, req.EventID)
	if err != nil {
		if err.Error() == "没有新闻被更新，请检查新闻ID是否正确" {
			utils.BadRequest(c, err.Error())
		} else {
			utils.InternalServerError(c, err.Error())
		}
		return
	}

	message := "News event association updated successfully"
	if req.EventID == nil {
		message = "News event association removed successfully"
	}

	utils.Success(c, gin.H{"message": message})
}

// GetNewsByEventID 根据事件ID获取关联的新闻
func (h *NewsHandler) GetNewsByEventID(c *gin.Context) {
	eventIDStr := c.Param("event_id")
	eventID, err := strconv.ParseUint(eventIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	newsList, err := h.newsService.GetNewsByEventID(uint(eventID))
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	utils.Success(c, newsResponses)
}

// GetHotNews 获取热门新闻
func (h *NewsHandler) GetHotNews(c *gin.Context) {
	// 获取limit参数，默认为10
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 10 // 限制最大100条
	}

	newsList, err := h.newsService.GetHotNews(limit)
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	utils.Success(c, newsResponses)
}
