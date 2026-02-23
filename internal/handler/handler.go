package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"easypanel/internal/config"
)

// ── AUTH ──────────────────────────────────────────────────────────

func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	user := config.FindUser(req.Username)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": user.Username,
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	tokenStr, err := token.SignedString([]byte(config.Main.JWTSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token error"})
		return
	}
	c.SetCookie("token", tokenStr, 7*24*3600, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"token":    tokenStr,
		"username": user.Username,
		"nickname": user.Nickname,
		"is_admin": user.IsAdmin,
	})
}

func Logout(c *gin.Context) {
	c.SetCookie("token", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── PANEL INFO (public) ───────────────────────────────────────────

func GetPanelInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"hostname":    config.Settings.Hostname,
		"logo":        config.Settings.Logo,
		"wallpaper":   config.Settings.Wallpaper,
		"clock":       config.Settings.Clock,
		"public_mode": config.Main.PublicMode,
	})
}

func GetApps(c *gin.Context) {
	c.JSON(http.StatusOK, config.Apps)
}

// ── APPS (auth required) ──────────────────────────────────────────

func CreateApp(c *gin.Context) {
	var app config.AppItem
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	b := make([]byte, 8)
	rand.Read(b)
	app.ID = hex.EncodeToString(b)
	if app.OpenType == "" {
		app.OpenType = "new_tab"
	}
	if app.IconType == "" {
		app.IconType = "text"
	}
	config.AddApp(app)
	if err := config.SaveApps(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	c.JSON(http.StatusOK, app)
}

func UpdateApp(c *gin.Context) {
	id := c.Param("id")
	var app config.AppItem
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !config.UpdateApp(id, app) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	config.SaveApps()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func DeleteApp(c *gin.Context) {
	id := c.Param("id")
	if !config.DeleteApp(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	config.SaveApps()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ReorderApps(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.ReorderApps(req.IDs)
	config.SaveApps()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── UPLOAD (auth required) ────────────────────────────────────────

func uploadFile(c *gin.Context, field, prefix string) {
	file, err := c.FormFile(field)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file"})
		return
	}
	ext := filepath.Ext(file.Filename)
	b := make([]byte, 8)
	rand.Read(b)
	filename := fmt.Sprintf("%s%s%s", prefix, hex.EncodeToString(b), ext)
	dst := filepath.Join(config.DataDir, "uploads", filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + filename})
}

func UploadImage(c *gin.Context)    { uploadFile(c, "image", "") }
func UploadWallpaper(c *gin.Context) { uploadFile(c, "wallpaper", "wp_") }
func UploadLogo(c *gin.Context)     { uploadFile(c, "logo", "logo_") }

// ── SETTINGS (auth required) ──────────────────────────────────────

func GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, config.Settings)
}

func UpdateSettings(c *gin.Context) {
	var s config.PanelSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Settings = &s
	config.SaveSettings()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func GetPublicMode(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"public_mode": config.Main.PublicMode})
}

func SetPublicMode(c *gin.Context) {
	var req struct {
		PublicMode bool `json:"public_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Main.PublicMode = req.PublicMode
	config.SaveMain()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ── ACCOUNT (auth required) ───────────────────────────────────────

func GetMe(c *gin.Context) {
	username := c.GetString("username")
	user := config.FindUser(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"username": user.Username,
		"nickname": user.Nickname,
		"is_admin": user.IsAdmin,
	})
}

func UpdatePassword(c *gin.Context) {
	username := c.GetString("username")
	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := config.FindUser(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	user.Password = string(hash)
	config.SaveMain()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func UpdateNickname(c *gin.Context) {
	username := c.GetString("username")
	var req struct{ Nickname string `json:"nickname"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := config.FindUser(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	user.Nickname = req.Nickname
	config.SaveMain()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ListUsers(c *gin.Context) {
	users := make([]gin.H, 0)
	for _, u := range config.Main.Users {
		users = append(users, gin.H{
			"username": u.Username,
			"nickname": u.Nickname,
			"is_admin": u.IsAdmin,
		})
	}
	c.JSON(http.StatusOK, users)
}

func CreateUser(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if config.FindUser(req.Username) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user exists"})
		return
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	config.Main.Users = append(config.Main.Users, config.User{
		Username: req.Username, Password: string(hash),
		Nickname: req.Nickname, IsAdmin: req.IsAdmin,
	})
	config.SaveMain()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func DeleteUser(c *gin.Context) {
	target := c.Param("username")
	me := c.GetString("username")
	if target == me {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete self"})
		return
	}
	for i, u := range config.Main.Users {
		if u.Username == target {
			config.Main.Users = append(config.Main.Users[:i], config.Main.Users[i+1:]...)
			config.SaveMain()
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

// ── CHECK AUTH (for public mode: returns whether logged in) ───────

func CheckAuth(c *gin.Context) {
	// Try cookie
	tokenStr, err := c.Cookie("token")
	if err != nil || tokenStr == "" {
		c.JSON(http.StatusOK, gin.H{"logged_in": false})
		return
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(config.Main.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusOK, gin.H{"logged_in": false})
		return
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"logged_in": false})
		return
	}
	username, _ := claims["sub"].(string)
	user := config.FindUser(username)
	if user == nil {
		c.JSON(http.StatusOK, gin.H{"logged_in": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"logged_in": true,
		"username":  user.Username,
		"nickname":  user.Nickname,
		"is_admin":  user.IsAdmin,
	})
}

// keep compiler happy
var _ = os.Getenv
