package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

// ── easypanel.yaml：端口、密钥、用户、公开模式 ──────────────────

type User struct {
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"-"`
	Nickname string `yaml:"nickname" json:"nickname"`
	IsAdmin  bool   `yaml:"is_admin" json:"is_admin"`
}

type MainConfig struct {
	Port             int       `yaml:"port"`
	JWTSecret        string    `yaml:"jwt_secret"`
	PublicMode       bool      `yaml:"public_mode"`
	CacheIntervalSec int       `yaml:"cache_interval_sec"` // 后台缓存刷新间隔（秒），默认 30
	Users            []User    `yaml:"users"`
	CreatedAt        time.Time `yaml:"created_at"`
}

// ── data/apps.json ───────────────────────────────────────────────

type AppItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`      // 保留兼容旧数据
	UrlLan    string `json:"url_lan"`  // 内网地址
	UrlWan    string `json:"url_wan"`  // 公网地址
	IconType  string `json:"icon_type"`
	IconText  string `json:"icon_text"`
	IconImage string `json:"icon_image"`
	OpenType  string `json:"open_type"`
	Order     int    `json:"order"`
}

// ── data/settings.json ───────────────────────────────────────────

type ClockDisplay struct {
	ShowTime    bool `json:"show_time"`
	ShowDate    bool `json:"show_date"`
	ShowWeekday bool `json:"show_weekday"`
	ShowLunar   bool `json:"show_lunar"`
	ShowSeconds bool `json:"show_seconds"`
	ShowYear    bool `json:"show_year"`
}

// DisplayConfig 保存一套显示样式（桌面端或移动端）
type DisplayConfig struct {
	HostnameSize int    `json:"hostname_size"`
	ClockSize    int    `json:"clock_size"`
	IconSize     int    `json:"icon_size"`
	AppNameSize  int    `json:"app_name_size"`
	IconRadius   int    `json:"icon_radius"`
	IconGap      int    `json:"icon_gap"`
	SidePadding  int    `json:"side_padding"`
	FontHostname string `json:"font_hostname"`
	FontClock    string `json:"font_clock"`
	FontAppname  string `json:"font_appname"`
	FontUI       string `json:"font_ui"`
}

type PanelSettings struct {
	Hostname     string       `json:"hostname"`
	Logo         string       `json:"logo"`
	Wallpaper    string       `json:"wallpaper"`
	Clock        ClockDisplay `json:"clock"`
	Theme        string       `json:"theme"`
	Language     string       `json:"language"`
	// 旧版顶层字段保留（兼容未迁移数据），新版以 Desktop/Mobile 为准
	HostnameSize int          `json:"hostname_size"`
	ClockSize    int          `json:"clock_size"`
	IconSize     int          `json:"icon_size"`
	AppNameSize  int          `json:"app_name_size"`
	IconRadius   int          `json:"icon_radius"`
	IconGap      int          `json:"icon_gap"`
	SidePadding  int          `json:"side_padding"`
	FontHostname string       `json:"font_hostname"`
	FontClock    string       `json:"font_clock"`
	FontAppname  string       `json:"font_appname"`
	FontUI       string       `json:"font_ui"`
	// 桌面端 / 移动端独立样式（0 表示未设置，回退到顶层旧字段）
	Desktop      *DisplayConfig `json:"desktop,omitempty"`
	Mobile       *DisplayConfig `json:"mobile,omitempty"`
	NetworkMode  string       `json:"network_mode"` // "lan" or "wan"
	ShowAppName     bool `json:"show_app_name"`
	FeatureSysInfo  bool `json:"feature_sysinfo"`
	FeatureProcess  bool `json:"feature_process"`
	FeatureSystemd  bool `json:"feature_systemd"`
	FeatureDocker   bool `json:"feature_docker"`
}

// ── 全局状态 ─────────────────────────────────────────────────────

var (
	Main     *MainConfig
	Apps     []AppItem
	Settings *PanelSettings
	DataDir  = "data"
)

// ── Init ─────────────────────────────────────────────────────────

func Init() error {
	os.MkdirAll(DataDir, 0755)
	os.MkdirAll(DataDir+"/uploads", 0755)
	if err := loadMain(); err != nil {
		return err
	}
	if err := loadApps(); err != nil {
		return err
	}
	return loadSettings()
}

// ── easypanel.yaml ────────────────────────────────────────────────

const configDir = "config"
const configPath = "config/easypanel.yaml"

func loadMain() error {
	os.MkdirAll(configDir, 0755)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// migrate from old location if exists
		if _, err2 := os.Stat("easypanel.yaml"); err2 == nil {
			data, _ := os.ReadFile("easypanel.yaml")
			os.WriteFile(configPath, data, 0600)
			os.Rename("easypanel.yaml", "easypanel.yaml.bak")
		} else {
			return createDefaultMain()
		}
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	Main = &MainConfig{}
	return yaml.Unmarshal(data, Main)
}

func createDefaultMain() error {
	secret := make([]byte, 32)
	rand.Read(secret)
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	Main = &MainConfig{
		Port:       3088,
		JWTSecret:  hex.EncodeToString(secret),
		PublicMode: false,
		Users:      []User{{Username: "admin", Password: string(hash), Nickname: "Admin", IsAdmin: true}},
		CreatedAt:  time.Now(),
	}
	return saveMain()
}

func saveMain() error {
	os.MkdirAll(configDir, 0755)
	data, err := yaml.Marshal(Main)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

func SaveMain() error { return saveMain() }

// ── data/apps.json ────────────────────────────────────────────────

func loadApps() error {
	path := DataDir + "/apps.json"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		Apps = []AppItem{}
		return saveApps()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &Apps)
}

func saveApps() error {
	data, err := json.MarshalIndent(Apps, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(DataDir+"/apps.json", data, 0644)
}

func SaveApps() error { return saveApps() }

// ── data/settings.json ────────────────────────────────────────────

func loadSettings() error {
	path := DataDir + "/settings.json"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		Settings = &PanelSettings{
			Hostname:     "EasyPanel",
			Wallpaper:    "https://images.unsplash.com/photo-1579546929518-9e396f3cc809?w=1920&q=80",
			Theme:        "purple-pink",
			Language:     "zh",
			HostnameSize: 70,
			ClockSize:    24,
			IconSize:     64,
			AppNameSize:  14,
			IconRadius:   25,
			IconGap:      22,
			SidePadding:  52,
			FontHostname: "system",
			FontClock:    "system",
			FontAppname:  "system",
			FontUI:       "system",
			ShowAppName:  true,
			Desktop: &DisplayConfig{
				HostnameSize: 70, ClockSize: 24, IconSize: 64, AppNameSize: 14,
				IconRadius: 25, IconGap: 22, SidePadding: 52,
				FontHostname: "system", FontClock: "system", FontAppname: "system", FontUI: "system",
			},
			Mobile: &DisplayConfig{
				HostnameSize: 48, ClockSize: 18, IconSize: 54, AppNameSize: 12,
				IconRadius: 25, IconGap: 16, SidePadding: 20,
				FontHostname: "system", FontClock: "system", FontAppname: "system", FontUI: "system",
			},
			FeatureSysInfo: true,
			Clock: ClockDisplay{
				ShowTime: true, ShowDate: true, ShowWeekday: true,
				ShowLunar: false, ShowSeconds: false,
			},
		}
		return saveSettings()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	Settings = &PanelSettings{}
	return json.Unmarshal(data, Settings)
}

func saveSettings() error {
	data, err := json.MarshalIndent(Settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(DataDir+"/settings.json", data, 0644)
}

func SaveSettings() error { return saveSettings() }

// ── User helpers ──────────────────────────────────────────────────

func FindUser(username string) *User {
	for i := range Main.Users {
		if Main.Users[i].Username == username {
			return &Main.Users[i]
		}
	}
	return nil
}

// ── App helpers ───────────────────────────────────────────────────

func AddApp(app AppItem) {
	app.Order = len(Apps)
	Apps = append(Apps, app)
}

func UpdateApp(id string, updated AppItem) bool {
	for i := range Apps {
		if Apps[i].ID == id {
			updated.ID = id
			updated.Order = Apps[i].Order
			Apps[i] = updated
			return true
		}
	}
	return false
}

func DeleteApp(id string) bool {
	for i, a := range Apps {
		if a.ID == id {
			Apps = append(Apps[:i], Apps[i+1:]...)
			return true
		}
	}
	return false
}

func ReorderApps(ids []string) {
	appMap := make(map[string]AppItem)
	for _, a := range Apps {
		appMap[a.ID] = a
	}
	ordered := make([]AppItem, 0, len(Apps))
	for i, id := range ids {
		if a, ok := appMap[id]; ok {
			a.Order = i
			ordered = append(ordered, a)
		}
	}
	Apps = ordered
}
