package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"easypanel/internal/config"
	"easypanel/internal/handler"
	"easypanel/internal/middleware"
)

//go:embed web/templates/index.html
var indexHTML embed.FS

func main() {
	gin.SetMode(gin.ReleaseMode)

	// Init config
	if err := config.Init(); err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	fmt.Printf("\n🚀 EasyPanel is running!\n")
	fmt.Printf("   URL: http://localhost:%d\n", config.Current.Port)
	fmt.Printf("   Default credentials: admin / admin\n\n")

	r := gin.Default()

	// Serve uploaded files
	os.MkdirAll("uploads", 0755)
	r.Static("/uploads", "./uploads")

	// Serve index HTML for all frontend routes
	indexContent, err := fs.ReadFile(indexHTML, "web/templates/index.html")
	if err != nil {
		log.Fatal("Failed to read index.html:", err)
	}

	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexContent)
	})

	// Public API
	r.POST("/api/login", handler.Login)

	// Protected API
	auth := r.Group("/api", middleware.Auth())
	{
		auth.POST("/logout", handler.Logout)
		auth.GET("/panel", handler.GetPanelInfo)

		// Apps
		auth.GET("/apps", handler.GetApps)
		auth.POST("/apps", handler.CreateApp)
		auth.PUT("/apps/:id", handler.UpdateApp)
		auth.DELETE("/apps/:id", handler.DeleteApp)
		auth.POST("/apps/reorder", handler.ReorderApps)

		// Upload
		auth.POST("/upload", handler.UploadImage)
		auth.POST("/upload/wallpaper", handler.UploadWallpaper)
		auth.POST("/upload/logo", handler.UploadLogo)

		// Settings
		auth.GET("/settings", handler.GetSettings)
		auth.PUT("/settings", handler.UpdateSettings)

		// Account
		auth.GET("/me", handler.GetMe)
		auth.PUT("/me/nickname", handler.UpdateNickname)
		auth.PUT("/me/password", handler.UpdatePassword)

		// User management (admin)
		auth.GET("/users", handler.ListUsers)
		auth.POST("/users", handler.CreateUser)
		auth.DELETE("/users/:username", handler.DeleteUser)
	}

	addr := fmt.Sprintf(":%d", config.Current.Port)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
