package main

import (
	"log"
	"net/http"

	"realtime_chat_platform/internal/database"
	"realtime_chat_platform/internal/handlers"
	"realtime_chat_platform/internal/middleware"
	"realtime_chat_platform/internal/websocket"

	"github.com/gin-gonic/gin"
)

func main() {
	// инициализация базы данных
	database.InitDB()

	// запуск WebSocket хаба
	go websocket.GlobalHub.Run()

	r := gin.Default()

	// сервер статических файлов
	r.Static("/static", "./web/static")
	r.LoadHTMLGlob("web/templates/*")

	// маршруты
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "Realtime Chat Platform",
		})
	})

	r.GET("/profile", func(c *gin.Context) {
		c.HTML(http.StatusOK, "profile.html", gin.H{
			"title": "Профиль пользователя",
		})
	})

	// маршруты API
	api := r.Group("/api")
	{
		api.POST("/register", handlers.RegisterHandler)
		api.POST("/login", handlers.LoginHandler)
		api.GET("/messages", handlers.GetMessageHistory)
		api.GET("/users/online", handlers.GetOnlineUsers)
		api.GET("/ws", func(c *gin.Context) {
			websocket.WebSocketHandler(c.Writer, c.Request)
		})

		// маршруты профиля
		profile := api.Group("/profile")
		profile.Use(middleware.AuthMiddleware())
		{
			profile.GET("/", handlers.GetProfileHandler)
			profile.PUT("/", handlers.UpdateProfileHandler)
			profile.PUT("/password", handlers.ChangePasswordHandler)
			profile.POST("/avatar", handlers.UploadAvatarHandler)
		}

		// маршрут публичного профиля пользователя
		api.GET("/users/:username/profile", handlers.GetUserProfileHandler)
	}

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
