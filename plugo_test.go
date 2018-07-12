package plugo

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	bot := NewBot()
	if err := bot.loadConfig("./examples/bot/config.toml"); err != nil {
		t.Errorf("Error loading config - %s", err.Error())
	}
}

func TestCreateSession(t *testing.T) {
	bot := NewBot()
	bot.Config.Token = "test"
	if err := bot.createSession(); err != nil {
		t.Errorf("Error creating session - %s", err.Error())
	}
}

func TestLoadPlugins(t *testing.T) {
	bot := NewBot()
	bot.Config.PluginsDir = "./examples/plugins"
	if err := bot.loadPlugins(); err != nil {
		t.Errorf("Error loading plugins - %s", err.Error())
	}
}
