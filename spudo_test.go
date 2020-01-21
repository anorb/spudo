package spudo

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	bot := newSpudo()
	if err := bot.loadConfig("./examples/bot/config.toml"); err != nil {
		t.Errorf("Error loading config - %s", err.Error())
	}
}

func TestCreateSession(t *testing.T) {
	bot := newSpudo()
	bot.Config.Token = "test"
	bot.logger = newLogger()
	var err error
	if bot.SpudoSession, err = newSpudoSession(bot.Config.Token, bot.logger); err != nil {
		t.Errorf("Error creating session - %s", err.Error())
	}
}
