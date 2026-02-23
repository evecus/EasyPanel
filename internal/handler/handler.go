package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"easypanel/internal/config"
)

// ===== AUTH =====

func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	user := config.Current.FindUser(req.Username)
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
	tokenStr, err := token.SignedString([]byte(config.Current.JWTSecret))
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

// ===== PANEL INFO =====

func GetPanelInfo(c *gin.Context) {
	hostname, _ := os.Hostname()
	c.JSON(http.StatusOK, gin.H{
		"title":     config.Current.Panel.Title,
		"logo":      config.Current.Panel.Logo,
		"wallpaper": config.Current.Panel.Wallpaper,
		"hostname":  config.Current.Panel.Hostname,
		"sys_hostname": hostname,
	})
}

// ===== APPS =====

func GetApps(c *gin.Context) {
	c.JSON(http.StatusOK, config.Current.Apps)
}

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

	config.Current.AddApp(app)
	if err := config.Save(); err != nil {
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
	if !config.Current.UpdateApp(id, app) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	config.Save()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func DeleteApp(c *gin.Context) {
	id := c.Param("id")
	if !config.Current.DeleteApp(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	config.Save()
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
	config.Current.ReorderApps(req.IDs)
	config.Save()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ===== UPLOAD =====

func UploadImage(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file"})
		return
	}

	ext := filepath.Ext(file.Filename)
	b := make([]byte, 8)
	rand.Read(b)
	filename := fmt.Sprintf("%s%s", hex.EncodeToString(b), ext)

	uploadDir := "uploads"
	os.MkdirAll(uploadDir, 0755)

	dst := filepath.Join(uploadDir, filename)
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "open failed"})
		return
	}
	defer src.Close()

	out, err := os.Create(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	defer out.Close()
	io.Copy(out, src)

	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + filename})
}

// ===== SETTINGS =====

func GetSettings(c *gin.Context) {
	c.JSON(http.StatusOK, config.Current.Panel)
}

func UpdateSettings(c *gin.Context) {
	var s config.PanelSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config.Current.Panel = s
	config.Save()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ===== ACCOUNT =====

func GetMe(c *gin.Context) {
	username := c.GetString("username")
	user := config.Current.FindUser(username)
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

	user := config.Current.FindUser(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "wrong password"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}
	user.Password = string(hash)
	config.Save()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func UpdateNickname(c *gin.Context) {
	username := c.GetString("username")
	var req struct {
		Nickname string `json:"nickname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user := config.Current.FindUser(username)
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	user.Nickname = req.Nickname
	config.Save()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func ListUsers(c *gin.Context) {
	users := make([]gin.H, 0)
	for _, u := range config.Current.Users {
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

	if config.Current.FindUser(req.Username) != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user exists"})
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	config.Current.Users = append(config.Current.Users, config.User{
		Username: req.Username,
		Password: string(hash),
		Nickname: req.Nickname,
		IsAdmin:  req.IsAdmin,
	})
	config.Save()
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func DeleteUser(c *gin.Context) {
	target := c.Param("username")
	me := c.GetString("username")
	if target == me {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete self"})
		return
	}
	for i, u := range config.Current.Users {
		if u.Username == target {
			config.Current.Users = append(config.Current.Users[:i], config.Current.Users[i+1:]...)
			config.Save()
			c.JSON(http.StatusOK, gin.H{"ok": true})
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
}

func UploadWallpaper(c *gin.Context) {
	file, err := c.FormFile("wallpaper")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file"})
		return
	}

	ext := filepath.Ext(file.Filename)
	b := make([]byte, 8)
	rand.Read(b)
	filename := fmt.Sprintf("wallpaper_%s%s", hex.EncodeToString(b), ext)

	uploadDir := "uploads"
	os.MkdirAll(uploadDir, 0755)

	dst := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + filename})
}

func UploadLogo(c *gin.Context) {
	file, err := c.FormFile("logo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file"})
		return
	}

	ext := filepath.Ext(file.Filename)
	b := make([]byte, 8)
	rand.Read(b)
	filename := fmt.Sprintf("logo_%s%s", hex.EncodeToString(b), ext)

	uploadDir := "uploads"
	os.MkdirAll(uploadDir, 0755)

	dst := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": "/uploads/" + filename})
}
