package main

import (
	"net/http"

	"hidevideo/backend/config"
	"hidevideo/backend/database"
	"hidevideo/backend/handlers"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	if err := database.Init(); err != nil {
		panic(err)
	}

	// 初始化 Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// 设置静态文件服务
	// 视频封面
	r.Static("/covers", config.ServerConfig.StaticPath)

	// 设置 Session
	store := cookie.NewStore([]byte(config.SessionConfig.Secret))
	store.Options(sessions.Options{
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
	})
	r.Use(sessions.Sessions("hidevideo_session", store))

	// API 路由
	api := r.Group("/api")
	{
		// 认证相关
		auth := api.Group("/auth")
		{
			auth.POST("/login", handlers.Login)
			auth.POST("/logout", handlers.Logout)
			auth.GET("/check", handlers.CheckAuth)
			auth.GET("/captcha", handlers.GetCaptcha)
		}

		// 系统设置
		settings := api.Group("/settings")
		{
			settings.GET("/login-protection", handlers.GetLoginProtection)
			settings.POST("/login-protection", handlers.SetLoginProtection)
		}

		// 需要登录的 API
		protected := api.Group("")
		protected.Use(handlers.AuthMiddleware())
		{
			// 视频库管理
			libraries := protected.Group("/libraries")
			{
				libraries.GET("", handlers.GetLibraries)
				libraries.POST("", handlers.AddLibrary)
				libraries.DELETE("/:id", handlers.DeleteLibrary)
				libraries.POST("/:id/scan", handlers.ScanLibrary)
				libraries.POST("/:id/cover", handlers.GenerateCovers)
				libraries.POST("/clean-invalid", handlers.CleanInvalidIndex)
				libraries.GET("/:id/path", handlers.GetLibraryPath)
				libraries.GET("/:id/files", handlers.ListLibraryFiles)
				libraries.POST("/:id/icon", handlers.GenerateIcon)
			}

			// 视频管理
			videos := protected.Group("/videos")
			{
				videos.GET("", handlers.GetVideos)
				videos.GET("/folders", handlers.GetFolderTree)
				videos.GET("/by-path", handlers.GetVideoByPath)
				videos.GET("/:id", handlers.GetVideo)
				videos.GET("/:id/stream", handlers.StreamVideo)
				videos.PUT("/:id/rating", handlers.UpdateRating)
				videos.PUT("/:id/filename", handlers.UpdateVideoFilename)
				videos.POST("/:id/play", handlers.IncrementPlayCount)
				videos.DELETE("/:id", handlers.DeleteVideo)

				// 视频标签
				videos.GET("/:id/tags", handlers.GetVideoTags)
				videos.POST("/:id/tags", handlers.AddVideoTag)
				videos.DELETE("/:id/tags/:tagId", handlers.RemoveVideoTag)

				// 视频评论
				videos.GET("/:id/comments", handlers.GetComments)
				videos.POST("/:id/comments", handlers.AddComment)
			}

			// 评论管理
			protected.DELETE("/comments/:id", handlers.DeleteComment)

			// 标签管理
			tags := protected.Group("/tags")
			{
				tags.GET("", handlers.GetTags)
				tags.POST("", handlers.AddTag)
				tags.PUT("/reorder", handlers.ReorderTags)
				tags.PUT("/:id", handlers.UpdateTag)
				tags.DELETE("/:id", handlers.DeleteTag)
			}

			// 演员管理
			actors := protected.Group("/actors")
			{
				actors.GET("", handlers.GetActors)
				actors.POST("", handlers.AddActor)
				actors.PUT("/reorder", handlers.ReorderActors)
				actors.PUT("/:id", handlers.UpdateActor)
				actors.DELETE("/:id", handlers.DeleteActor)

				// 演员参演的视频
				actors.GET("/:id/videos", handlers.GetActorVideos)
			}

			// 视频演员
			videosActor := protected.Group("/videos")
			{
				videosActor.GET("/:id/actors", handlers.GetVideoActors)
				videosActor.POST("/:id/actors", handlers.AddVideoActor)
				videosActor.DELETE("/:id/actors/:actorId", handlers.RemoveVideoActor)
			}

			// 用户管理
			users := protected.Group("/users")
			{
				users.GET("", handlers.GetUsers)
				users.POST("", handlers.AddUser)
				users.DELETE("/:id", handlers.DeleteUser)
				users.PUT("/:id/password", handlers.AdminUpdateUserPassword)
				users.PUT("/:id/info", handlers.AdminUpdateUserInfo)
				users.PUT("/password", handlers.UpdateUserPassword)
				users.PUT("/info", handlers.UpdateUserInfo)
				users.GET("/me", handlers.GetCurrentUser)
			}
		}

		// 前端路由 - SPA fallback
		distDir := http.Dir("./frontend/dist")
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			// 静态资源使用 FileServer 确保正确的 MIME 类型
			if len(path) > 1 && (path[:7] == "/assets" || path[:8] == "/static" || path == "/favicon.svg") {
				// 检查文件是否存在
				f, err := distDir.Open(path)
				if err != nil {
					// 文件不存在，返回 404
					c.Status(404)
					return
				}
				f.Close()
				http.FileServer(distDir).ServeHTTP(c.Writer, c.Request)
				return
			}
			// 其他路径返回 index.html (SPA fallback)
			c.File("./frontend/dist/index.html")
		})
	}

	// 启动服务器
	r.Run(":" + config.ServerConfig.Port)
}
