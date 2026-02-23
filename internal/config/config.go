package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"` // bcrypt hashed
	Nickname string `yaml:"nickname"`
	IsAdmin  bool   `yaml:"is_admin"`
}

type AppItem struct {
	ID          string `yaml:"id"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	URL         string `yaml:"url"`
	IconType    string `yaml:"icon_type"` // "text" or "image"
	IconText    string `yaml:"icon_text,omitempty"`
	IconImage   string `yaml:"icon_image,omitempty"` // URL or local path
	Group       string `yaml:"group,omitempty"`
	Order       int    `yaml:"order"`
	OpenType    string `yaml:"open_type"` // "new_tab" or "current"
}

type PanelSettings struct {
	Title     string `yaml:"title"`
	Logo      string `yaml:"logo,omitempty"`
	Wallpaper string `yaml:"wallpaper,omitempty"` // URL or local path
	Hostname  string `yaml:"hostname"`
}

type Config struct {
	Port      int           `yaml:"port"`
	JWTSecret string        `yaml:"jwt_secret"`
	Users     []User        `yaml:"users"`
	Apps      []AppItem     `yaml:"apps"`
	Panel     PanelSettings `yaml:"panel"`
	CreatedAt time.Time     `yaml:"created_at"`
}

var configPath = "easypanel.yaml"
var Current *Config

func Init() error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefault()
	}
	return load()
}

func createDefault() error {
	secret := make([]byte, 32)
	rand.Read(secret)

	hostname, _ := os.Hostname()

	hash, _ := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)

	cfg := &Config{
		Port:      3000,
		JWTSecret: hex.EncodeToString(secret),
		Users: []User{
			{
				Username: "admin",
				Password: string(hash),
				Nickname: "Admin",
				IsAdmin:  true,
			},
		},
		Apps: []AppItem{},
		Panel: PanelSettings{
			Title:    "EasyPanel",
			Hostname: hostname,
		},
		CreatedAt: time.Now(),
	}

	Current = cfg
	return save()
}

func load() error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return err
	}
	Current = cfg
	return nil
}

func save() error {
	data, err := yaml.Marshal(Current)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

func Save() error {
	return save()
}

func (c *Config) FindUser(username string) *User {
	for i := range c.Users {
		if c.Users[i].Username == username {
			return &c.Users[i]
		}
	}
	return nil
}

func (c *Config) AddApp(app AppItem) {
	app.Order = len(c.Apps)
	c.Apps = append(c.Apps, app)
}

func (c *Config) UpdateApp(id string, updated AppItem) bool {
	for i := range c.Apps {
		if c.Apps[i].ID == id {
			updated.ID = id
			updated.Order = c.Apps[i].Order
			c.Apps[i] = updated
			return true
		}
	}
	return false
}

func (c *Config) DeleteApp(id string) bool {
	for i, app := range c.Apps {
		if app.ID == id {
			c.Apps = append(c.Apps[:i], c.Apps[i+1:]...)
			return true
		}
	}
	return false
}

func (c *Config) ReorderApps(ids []string) {
	ordered := make([]AppItem, 0, len(c.Apps))
	appMap := make(map[string]AppItem)
	for _, app := range c.Apps {
		appMap[app.ID] = app
	}
	for i, id := range ids {
		if app, ok := appMap[id]; ok {
			app.Order = i
			ordered = append(ordered, app)
		}
	}
	c.Apps = ordered
}
