package handler

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"easypanel/internal/collector"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"easypanel/internal/config"
)

// AppVersion is set from main.go at startup via ldflags
var AppVersion = "dev"

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
	s := config.Settings
	hostnameSize := s.HostnameSize
	if hostnameSize == 0 {
		hostnameSize = 56
	}
	clockSize := s.ClockSize
	if clockSize == 0 {
		clockSize = 16
	}
	iconSize := s.IconSize
	if iconSize == 0 {
		iconSize = 78
	}
	appNameSize := s.AppNameSize
	if appNameSize == 0 {
		appNameSize = 12
	}
	iconRadius := s.IconRadius
	if iconRadius == 0 {
		iconRadius = 26
	}
	iconGap := s.IconGap
	if iconGap == 0 {
		iconGap = 22
	}
	sidePadding := s.SidePadding
	if sidePadding == 0 {
		sidePadding = 52
	}
	c.JSON(http.StatusOK, gin.H{
		"hostname":      s.Hostname,
		"logo":          s.Logo,
		"wallpaper":     s.Wallpaper,
		"clock":         s.Clock,
		"theme":         s.Theme,
		"language":      s.Language,
		"hostname_size": hostnameSize,
		"clock_size":    clockSize,
		"icon_size":     iconSize,
		"app_name_size": appNameSize,
		"icon_radius":   iconRadius,
		"icon_gap":      iconGap,
		"side_padding":  sidePadding,
		"font_hostname": s.FontHostname,
		"font_clock":    s.FontClock,
		"font_appname":  s.FontAppname,
		"font_ui":       s.FontUI,
		"public_mode":   config.Main.PublicMode,
		"network_mode":     s.NetworkMode,
		"feature_sysinfo":  s.FeatureSysInfo,
		"feature_process":  s.FeatureProcess,
		"feature_systemd":  s.FeatureSystemd,
		"feature_docker":   s.FeatureDocker,
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
	// 功能被关闭时，立即清除对应缓存
	if config.Settings != nil {
		if config.Settings.FeatureSystemd && !s.FeatureSystemd {
			collector.InvalidateSystemdCache()
		}
		if config.Settings.FeatureDocker && !s.FeatureDocker {
			collector.InvalidateDockerCache()
		}
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

// ── FETCH ICON ───────────────────────────────────────────────

func FetchIcon(c *gin.Context) {
	rawURL := strings.TrimSpace(c.Query("url"))
	if rawURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url required"})
		return
	}

	// 补全协议
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	// 先尝试抓 HTML 解析 <link rel="icon">
	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Get(rawURL)
	iconURL := ""
	if err == nil {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
		html := string(body)
		// 匹配 <link rel="icon/shortcut icon/apple-touch-icon" href="...">
		re := regexp.MustCompile(`(?i)<link[^>]+rel=["'](?:shortcut icon|icon|apple-touch-icon)["'][^>]+href=["']([^"']+)["']`)
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			iconURL = m[1]
		} else {
			// 反过来找 href 在前的情况
			re2 := regexp.MustCompile(`(?i)<link[^>]+href=["']([^"']+)["'][^>]+rel=["'](?:shortcut icon|icon|apple-touch-icon)["']`)
			if m2 := re2.FindStringSubmatch(html); len(m2) > 1 {
				iconURL = m2[1]
			}
		}
	}

	// 转为绝对路径
	base := parsed.Scheme + "://" + parsed.Host
	if iconURL != "" {
		if strings.HasPrefix(iconURL, "//") {
			iconURL = parsed.Scheme + ":" + iconURL
		} else if strings.HasPrefix(iconURL, "/") {
			iconURL = base + iconURL
		} else if !strings.HasPrefix(iconURL, "http") {
			iconURL = base + "/" + iconURL
		}
	} else {
		// fallback: /favicon.ico
		iconURL = base + "/favicon.ico"
	}

	c.JSON(http.StatusOK, gin.H{"icon": iconURL})
}

// ── SYSTEM MONITOR APIs ───────────────────────────────────────────

func GetSystemInfo(c *gin.Context)    { c.JSON(200, collector.GetSystemInfo()) }
func GetCPUStats(c *gin.Context)      { c.JSON(200, collector.GetCPUStats()) }
func GetMemoryStats(c *gin.Context)   { c.JSON(200, collector.GetMemoryStats()) }
func GetDiskStats(c *gin.Context)     { c.JSON(200, collector.GetDiskStats()) }
func GetNetworkStats(c *gin.Context)  { c.JSON(200, collector.GetNetworkStats()) }
func GetTemperatures(c *gin.Context)  { c.JSON(200, collector.GetTemperatures()) }

func GetMetricsAll(c *gin.Context) {
	c.JSON(200, collector.CollectAll())
}

func GetProcesses(c *gin.Context) {
	sortBy  := c.DefaultQuery("sort", "cpu")
	sortDir := c.DefaultQuery("dir", "desc")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	procs, err := collector.GetProcesses(sortBy, sortDir, limit)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, procs)
}

func KillProcess(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("pid"), 10, 32)
	if err != nil { c.JSON(400, gin.H{"error": "invalid pid"}); return }
	if err := collector.KillProcess(int32(pid)); err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"ok": true})
}

func GetContainers(c *gin.Context) {
	data, err := collector.GetContainersFromCache()
	if err != nil { c.JSON(200, []interface{}{}); return }
	c.JSON(200, data)
}

func ContainerAction(c *gin.Context) {
	id, action := c.Param("id"), c.Param("action")
	if !map[string]bool{"start": true, "stop": true, "restart": true}[action] {
		c.JSON(400, gin.H{"error": "invalid action"}); return
	}
	if err := collector.ContainerAction(id, action); err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	// 操作后让缓存失效，下次请求立即重新采集
	collector.InvalidateDockerCache()
	c.JSON(200, gin.H{"ok": true})
}

func GetContainerLogs(c *gin.Context) {
	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "200"))
	logs, err := collector.GetContainerLogs(c.Param("id"), lines)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"logs": logs})
}

func GetServices(c *gin.Context) {
	sortBy  := c.DefaultQuery("sort", "")
	sortDir := c.DefaultQuery("dir", "desc")
	data, err := collector.GetServicesFromCache(sortBy, sortDir)
	if err != nil { c.JSON(200, []interface{}{}); return }
	c.JSON(200, data)
}

func ServiceAction(c *gin.Context) {
	unit, action := c.Param("unit"), c.Param("action")
	if !map[string]bool{"start": true, "stop": true, "restart": true, "enable": true, "disable": true}[action] {
		c.JSON(400, gin.H{"error": "invalid action"}); return
	}
	if err := collector.ServiceAction(unit, action); err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	// 操作后让缓存失效，下次请求立即重新采集
	collector.InvalidateSystemdCache()
	c.JSON(200, gin.H{"ok": true})
}

func GetServiceLogs(c *gin.Context) {
	lines, _ := strconv.Atoi(c.DefaultQuery("lines", "200"))
	logs, err := collector.GetServiceLogs(c.Param("unit"), lines)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"logs": logs})
}

func GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": AppVersion})
}

// keep compiler happy
var _ = os.Getenv

func InspectContainer(c *gin.Context) {
	data, err := collector.InspectContainer(c.Param("id"))
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, data)
}

func GetComposeFile(c *gin.Context) {
	path := c.Query("path")
	if path == "" { c.JSON(400, gin.H{"error": "path required"}); return }
	content, err := collector.ReadComposeFile(path)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"content": content})
}

func ApplyCompose(c *gin.Context) {
	var body struct {
		Path        string `json:"path"`
		Content     string `json:"content"`
		ContainerID string `json:"container_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	log, err := collector.WriteAndApplyCompose(body.Path, body.Content, body.ContainerID)
	if err != nil { c.JSON(500, gin.H{"error": err.Error(), "log": log}); return }
	collector.InvalidateDockerCache()
	c.JSON(200, gin.H{"message": "重建成功", "log": log})
}

func PullUpdateContainer(c *gin.Context) {
	log, err := collector.PullAndUpdateContainer(c.Param("id"))
	if err != nil { c.JSON(500, gin.H{"error": err.Error(), "log": log}); return }
	collector.InvalidateDockerCache()
	c.JSON(200, gin.H{"log": log})
}
