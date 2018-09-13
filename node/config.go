package node

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

const (
	DefaultHTTPPort = 5050
	DefaultWSPort   = 5051
)

type Config struct {
	Name        string `toml:"-"`
	Version     string `toml:"-"`
	ModuleDir   string
	HTTPHost    string   `toml:",omitempty"`
	HTTPPort    int      `toml:",omitempty"`
	HTTPOrigins []string `toml:",omitempty"`
	HTTPModules []string `toml:",omitempty"`
	WSHost      string   `toml:",omitempty"`
	WSPort      int      `toml:",omitempty"`
	WSOrigins   []string `toml:",omitempty"`
	WSModules   []string `toml:",omitempty"`
}

// DefaultConfig contains reasonable default settings.
var DefaultConfig = Config{
	Name:      "task",
	Version:   "0.1",
	ModuleDir: DefaultDataDir(),
	HTTPHost:  "localhost",
	WSHost:    "localhost",
	HTTPPort:  DefaultHTTPPort,
	WSPort:    DefaultWSPort,
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "task", "modules")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "task", "modules")
		} else {
			return filepath.Join(home, ".task", "modules")
		}
	}
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
