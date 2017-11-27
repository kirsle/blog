package settings

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/kirsle/blog/core/jsondb"
)

// DB is a reference to the parent app's JsonDB object.
var DB *jsondb.DB

// Settings holds the global app settings.
type Settings struct {
	// Only gets set to true on save(), this determines whether
	// the site has ever been configured before.
	Initialized bool `json:"initialized"`

	Site struct {
		Title      string `json:"title"`
		AdminEmail string `json:"adminEmail"`
		URL        string `json:"url"`
	} `json:"site"`

	// Security-related settings.
	Security struct {
		SecretKey string `json:"secretKey"` // Session cookie secret key
		HashCost  int    `json:"hashCost"`  // Bcrypt hash cost for passwords
	} `json:"security"`

	// Redis settings for caching in JsonDB.
	Redis struct {
		Enabled bool   `json:"enabled"`
		Host    string `json:"host"`
		Port    int    `json:"port"`
		DB      int    `json:"db"`
		Prefix  string `json:"prefix"`
	} `json:"redis"`

	// Mail settings
	Mail struct {
		Enabled  bool   `json:"enabled"`
		Sender   string `json:"sender"`
		Host     string `json:"host"`
		Port     int    `json:"port"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"mail,omitempty"`
}

// Defaults returns default settings. The app initially sets this on
// startup before reading your site's saved settings (if available).
// Also this is used as a template when the user first configures their
// site.
func Defaults() *Settings {
	s := &Settings{}
	s.Site.Title = "Untitled Site"
	s.Security.HashCost = 14
	s.Security.SecretKey = RandomKey()
	s.Redis.Host = "localhost"
	s.Redis.Port = 6379
	s.Redis.DB = 0
	s.Mail.Host = "localhost"
	s.Mail.Port = 25
	return s
}

// RandomKey generates a random string to use for the site's secret key.
func RandomKey() string {
	keyLength := 32

	b := make([]byte, keyLength)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// Load the settings.
func Load() (*Settings, error) {
	s := &Settings{}
	err := DB.Get("app/settings", &s)
	return s, err
}

// Save the site settings.
func (s *Settings) Save() error {
	s.Initialized = true

	err := DB.Commit("app/settings", &s)
	if err != nil {
		return err
	}

	return nil
}
