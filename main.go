package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"easypanel/internal/config"
	"easypanel/internal/handler"
)

//go:embed web/templates/index.html
var indexHTML embed.FS

func main() {
	gin.SetMode(gin.ReleaseMode)

	if err := config.Init(); err != nil {
		log.Fatalf("Failed to init config: %v", err)
	}

	fmt.Printf("\n🚀 EasyPanel running on http://0.0.0.0:%d\n", config.Main.Port)
	fmt.Printf("   Default credentials: admin / admin\n\n")

	r := gin.Default()

	// Uploaded files served from data/uploads/
	r.Static("/uploads", "./"+config.DataDir+"/uploads")

	// Index HTML for all non-API routes
	indexContent, err := fs.ReadFile(indexHTML, "web/templates/index.html")
	if err != nil {
		log.Fatal("Failed to read index.html:", err)
	}
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexContent)
	})

	// ── Public API (no auth required) ────────────────────────────
	r.POST("/api/login", handler.Login)
	r.GET("/api/panel", handler.GetPanelInfo)
	r.GET("/api/apps", handler.GetApps)
	r.GET("/api/checkauth", handler.CheckAuth)

	// ── Protected API (JWT required) ─────────────────────────────
	auth := r.Group("/api", authMiddleware())
	{
		auth.POST("/logout", handler.Logout)
		auth.GET("/me", handler.GetMe)
		auth.PUT("/me/nickname", handler.UpdateNickname)
		auth.PUT("/me/password", handler.UpdatePassword)

		auth.POST("/apps", handler.CreateApp)
		auth.PUT("/apps/:id", handler.UpdateApp)
		auth.DELETE("/apps/:id", handler.DeleteApp)
		auth.POST("/apps/reorder", handler.ReorderApps)

		auth.POST("/upload", handler.UploadImage)
		auth.POST("/upload/wallpaper", handler.UploadWallpaper)
		auth.POST("/upload/logo", handler.UploadLogo)

		auth.GET("/settings", handler.GetSettings)
		auth.PUT("/settings", handler.UpdateSettings)
		auth.GET("/publicmode", handler.GetPublicMode)
		auth.PUT("/publicmode", handler.SetPublicMode)

		auth.GET("/users", handler.ListUsers)
		auth.POST("/users", handler.CreateUser)
		auth.DELETE("/users/:username", handler.DeleteUser)
	}

	addr := fmt.Sprintf("0.0.0.0:%d", config.Main.Port)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := ""
		if cookie, err := c.Cookie("token"); err == nil {
			tokenStr = cookie
		}
		if tokenStr == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				tokenStr = strings.TrimPrefix(auth, "Bearer ")
			}
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(config.Main.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		c.Set("username", claims["sub"])
		c.Next()
	}
}

// keep os imported
var _ = os.Getenv
