package api

import (
	"github.com/EasyPeek/EasyPeek-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// add cors middleware
	r.Use(middleware.CORSMiddleware())

	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "EasyPeek backend is running",
		})
	})

	// initialize handler
	userHandler := NewUserHandler()
	eventHandler := NewEventHandler()
	rssHandler := NewRSSHandler()
	adminHandler := NewAdminHandler()
	newsHandler := NewNewsHandler()

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
			// auth.POST("/refresh", userHandler.RefreshToken)  // TODO: 实现token刷新
			// auth.POST("/logout", userHandler.Logout)         // TODO: 实现登出
		}

		// user routes
		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)
			// 用户自删除账户
			user.DELETE("/me", userHandler.DeleteSelf)
		}

		// news routes
		news := v1.Group("/news")
		{
			// 公开路由 - 前端可以直接访问
			news.GET("", newsHandler.GetAllNews)                           // 获取所有新闻列表（带分页）
			news.GET("/:id", newsHandler.GetNewsByID)                      // 根据ID获取单条新闻
			news.GET("/search", newsHandler.SearchNews)                    // 搜索新闻
			news.GET("/hot", newsHandler.GetHotNews)                       // 获取热门新闻
			news.GET("/title", newsHandler.GetNewsByTitle)                 // 根据标题获取新闻
			news.GET("/category/:category", newsHandler.GetNewsByCategory) // 根据分类获取新闻
			news.GET("/unlinked", newsHandler.GetUnlinkedNews)             // 获取未关联事件的新闻
			news.GET("/event/:event_id", newsHandler.GetNewsByEventID)     // 根据事件ID获取新闻

			// 需要身份验证的路由
			authNews := news.Group("")
			authNews.Use(middleware.AuthMiddleware())
			{
				authNews.POST("", newsHandler.CreateNews)                                  // 创建新闻
				authNews.PUT("/:id", newsHandler.UpdateNews)                               // 更新新闻
				authNews.DELETE("/:id", newsHandler.DeleteNews)                            // 删除新闻
				authNews.PUT("/event-association", newsHandler.UpdateNewsEventAssociation) // 批量更新新闻事件关联
			}
		}

		// event routes
		events := v1.Group("/events")
		{
			// 公开路由
			events.GET("", eventHandler.GetEvents)
			events.GET("/hot", eventHandler.GetHotEvents)
			events.GET("/trending", eventHandler.GetTrendingEvents)
			events.GET("/categories", eventHandler.GetEventCategories)
			events.GET("/category/:category", eventHandler.GetEventsByCategory)
			events.GET("/tags", eventHandler.GetPopularTags)
			events.GET("/:id", eventHandler.GetEvent)
			events.GET("/:id/news", eventHandler.GetNewsByEventID)
			events.GET("/:id/stats", eventHandler.GetEventStats)
			events.GET("/status/:status", eventHandler.GetEventsByStatus)
			events.POST("/:id/view", eventHandler.IncrementViewCount)
			events.POST("/:id/share", eventHandler.ShareEvent)

			// 需要身份验证的路由
			authEvents := events.Group("")
			authEvents.Use(middleware.AuthMiddleware())
			{
				authEvents.POST("", eventHandler.CreateEvent)
				authEvents.PUT("/:id", eventHandler.UpdateEvent)
				authEvents.DELETE("/:id", eventHandler.DeleteEvent)
				authEvents.POST("/:id/like", eventHandler.LikeEvent)
				authEvents.POST("/:id/comment", eventHandler.AddComment)
			}

			// 管理员专用路由
			adminEvents := events.Group("")
			adminEvents.Use(middleware.AuthMiddleware())
			adminEvents.Use(middleware.RoleMiddleware(middleware.RoleAdmin))
			{
				adminEvents.PUT("/:id/tags", eventHandler.UpdateEventTags)
				adminEvents.POST("/generate", eventHandler.GenerateEventsFromNews)
			}

			// 系统内部路由（需要系统权限或管理员权限）
			systemEvents := events.Group("")
			systemEvents.Use(middleware.AuthMiddleware())
			systemEvents.Use(middleware.RequireSystemOrAdmin())
			{
				systemEvents.PUT("/:id/hotness", eventHandler.UpdateEventHotness)
			}
		}

		// RSS routes
		rss := v1.Group("/rss")
		{
			// 公开路由
			rss.GET("/news", rssHandler.GetNews)
			rss.GET("/news/hot", rssHandler.GetHotNews)
			rss.GET("/news/latest", rssHandler.GetLatestNews)
			rss.GET("/news/category/:category", rssHandler.GetNewsByCategory)
			rss.GET("/news/:id", rssHandler.GetNewsItem)

			// 管理员路由
			adminRSS := rss.Group("")
			adminRSS.Use(middleware.AuthMiddleware())
			adminRSS.Use(middleware.RoleMiddleware(middleware.RoleAdmin))
			{
				adminRSS.GET("/sources", rssHandler.GetRSSSources)
				adminRSS.POST("/sources", rssHandler.CreateRSSSource)
				adminRSS.PUT("/sources/:id", rssHandler.UpdateRSSSource)
				adminRSS.DELETE("/sources/:id", rssHandler.DeleteRSSSource)
				adminRSS.POST("/sources/:id/fetch", rssHandler.FetchRSSFeed)
				adminRSS.POST("/fetch-all", rssHandler.FetchAllRSSFeeds)
			}
		}

		// admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.AdminAuthMiddleware())
		{
			// 系统统计
			admin.GET("/stats", adminHandler.GetSystemStats)

			// 用户管理
			users := admin.Group("/users")
			{
				users.GET("", adminHandler.GetAllUsers)          // 获取所有用户（带过滤）
				users.GET("/active", userHandler.GetActiveUsers) // 获取活跃用户（保持兼容）
				users.GET("/:id", adminHandler.GetUserByID)      // 获取指定用户
				users.PUT("/:id", adminHandler.UpdateUser)       // 更新用户信息
				users.DELETE("/:id", adminHandler.DeleteUser)    // 管理员删除用户（硬删除）
				// 保留原有的单独角色和状态更新接口
				users.PUT("/:id/role", userHandler.UpdateUserRole)     // 更新用户角色
				users.PUT("/:id/status", userHandler.UpdateUserStatus) // 更新用户状态
			}

			// 事件管理
			events := admin.Group("/events")
			{
				events.GET("", adminHandler.GetAllEvents)       // 获取所有事件
				events.PUT("/:id", adminHandler.UpdateEvent)    // 更新事件
				events.DELETE("/:id", adminHandler.DeleteEvent) // 删除事件
			}

			// 新闻管理
			news := admin.Group("/news")
			{
				news.GET("", adminHandler.GetAllNews)        // 获取所有新闻
				news.PUT("/:id", adminHandler.UpdateNews)    // 更新新闻
				news.DELETE("/:id", adminHandler.DeleteNews) // 删除新闻
			}

			// RSS源管理
			rssAdmin := admin.Group("/rss-sources")
			{
				rssAdmin.GET("", adminHandler.GetAllRSSSources)            // 获取所有RSS源
				rssAdmin.POST("", adminHandler.CreateRSSSource)            // 创建RSS源
				rssAdmin.PUT("/:id", adminHandler.UpdateRSSSource)         // 更新RSS源
				rssAdmin.DELETE("/:id", adminHandler.DeleteRSSSource)      // 删除RSS源
				rssAdmin.POST("/:id/fetch", adminHandler.FetchRSSFeed)     // 手动抓取RSS源
				rssAdmin.POST("/fetch-all", adminHandler.FetchAllRSSFeeds) // 抓取所有RSS源
			}
		}

		// 系统管理路由（需要系统权限）
		system := v1.Group("/system")
		system.Use(middleware.AuthMiddleware())
		system.Use(middleware.RoleMiddleware(middleware.RoleSystem))
		{
			// 系统级用户管理
			systemUsers := system.Group("/users")
			{
				systemUsers.PUT("/:id/role", userHandler.UpdateUserRole) // 系统级角色更新
			}
		}
	}

	return r
}
