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
	Port       int       `yaml:"port"`
	JWTSecret  string    `yaml:"jwt_secret"`
	PublicMode bool      `yaml:"public_mode"`
	Users      []User    `yaml:"users"`
	CreatedAt  time.Time `yaml:"created_at"`
}

// ── data/apps.json ───────────────────────────────────────────────

type AppItem struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
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
}

type PanelSettings struct {
	Hostname  string       `json:"hostname"`
	Logo      string       `json:"logo"`
	Wallpaper string       `json:"wallpaper"`
	Clock     ClockDisplay `json:"clock"`
	Theme     string       `json:"theme"`
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

func loadMain() error {
	const path = "easypanel.yaml"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return createDefaultMain()
	}
	data, err := os.ReadFile(path)
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
		Port:       3000,
		JWTSecret:  hex.EncodeToString(secret),
		PublicMode: false,
		Users:      []User{{Username: "admin", Password: string(hash), Nickname: "Admin", IsAdmin: true}},
		CreatedAt:  time.Now(),
	}
	return saveMain()
}

func saveMain() error {
	data, err := yaml.Marshal(Main)
	if err != nil {
		return err
	}
	return os.WriteFile("easypanel.yaml", data, 0600)
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
			Hostname:  "EasyPanel",
			Wallpaper: "https://images.unsplash.com/photo-1579546929518-9e396f3cc809?w=1920&q=80",
			Theme:     "purple-pink",
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
