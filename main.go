package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"easypanel/internal/config"
	"easypanel/internal/handler"
)

// Embed the Vite build output.
// Run `cd frontend && npm run build` first to generate web/dist.
//go:embed web/dist
var distFS embed.FS

func main() {
	if exe, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exe)
		if real, err2 := filepath.EvalSymlinks(exeDir); err2 == nil {
			exeDir = real
		}
		if err3 := os.Chdir(exeDir); err3 != nil {
			log.Printf("Warning: could not chdir to %s: %v", exeDir, err3)
		}
	}

	gin.SetMode(gin.ReleaseMode)

	if err := config.Init(); err != nil {
		log.Fatalf("Failed to init config: %v", err)
	}

	fmt.Printf("\n🚀 EasyPanel running on http://0.0.0.0:%d\n", config.Main.Port)

	r := gin.Default()

	r.Static("/uploads", "./"+config.DataDir+"/uploads")

	// ── Public API ─────────────────────────────────────────────
	r.POST("/api/login", handler.Login)
	r.GET("/api/panel", handler.GetPanelInfo)
	r.GET("/api/apps", handler.GetApps)
	r.GET("/api/checkauth", handler.CheckAuth)
	r.GET("/api/fetch-icon", handler.FetchIcon)

	// ── Protected API ──────────────────────────────────────────
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

	// ── Frontend (Vite SPA) ────────────────────────────────────
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatal("Failed to sub distFS:", err)
	}
	staticHandler := http.FileServer(http.FS(sub))

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		// Serve real static files (JS chunks, CSS, images, icons)
		if strings.HasPrefix(path, "/assets/") ||
			path == "/favicon.ico" ||
			path == "/favicon.svg" ||
			path == "/vite.svg" {
			staticHandler.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA fallback
		indexFile, err := distFS.ReadFile("web/dist/index.html")
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexFile)
	})

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
			a := c.GetHeader("Authorization")
			if strings.HasPrefix(a, "Bearer ") {
				tokenStr = strings.TrimPrefix(a, "Bearer ")
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

var _ = os.Getenv
