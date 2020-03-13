package spudo

import (
	"errors"
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron/v3"
)

// Config contains all options for the config file
type Config struct {
	Token                 string
	CommandPrefix         string
	CooldownTimer         int
	CooldownMessage       string
	UnknownCommandMessage string
	AudioEnabled          bool
	RESTEnabled           bool
	RESTPort              string
}

// Spudo contains everything about the bot itself
type Spudo struct {
	sync.Mutex
	CommandMutex sync.Mutex
	*SpudoSession
	Config        Config
	CooldownList  map[string]time.Time
	TimersStarted bool
	logger        *spudoLogger

	spudoCommands map[string]*spudoCommand

	commands         map[string]*command
	startupPlugins   []*startupPlugin
	timedMessages    []*timedMessage
	userReactions    []*userReaction
	messageReactions []*messageReaction

	audioSessions map[string]*spAudio
}

type unknownCommand string

// Initialize will initialize everything Spudo needs to run.
func Initialize() *Spudo {
	sp := newSpudo()

	configPath := flag.String("config", "./config.toml", "TODO")
	flag.Parse()

	// Check if config exists, if it doesn't use
	// createMinimalConfig to generate one.
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		sp.logger.info("Config not detected, creating minimal config...")
		if err := sp.createMinimalConfig(); err != nil {
			sp.logger.fatal("Failed to create minimal config", err)
		}
	}

	if err := sp.loadConfig(*configPath); err != nil {
		sp.logger.fatal(err.Error())
	}

	return sp
}

// newSpudo setups up the logger and maps for the plugins.
func newSpudo() *Spudo {
	sp := &Spudo{}
	sp.CooldownList = make(map[string]time.Time)
	sp.logger = newLogger()
	sp.commands = make(map[string]*command)
	sp.timedMessages = make([]*timedMessage, 0)
	sp.userReactions = make([]*userReaction, 0)
	sp.messageReactions = make([]*messageReaction, 0)
	sp.spudoCommands = make(map[string]*spudoCommand)
	return sp
}

// Returns the default config settings for the bot
func getDefaultConfig() Config {
	return Config{
		Token:                 "",
		CommandPrefix:         "!",
		CooldownTimer:         10,
		CooldownMessage:       "Too many commands at once!",
		UnknownCommandMessage: "Invalid command!",
	}
}

// createMinimalConfig prompts the user to enter a Token and for the
// Config. This is used if no config is found.
func (sp *Spudo) createMinimalConfig() error {
	sp.Config = getDefaultConfig()
	path := "./config.toml"

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	err = toml.NewEncoder(f).Encode(sp.Config)
	if err != nil {
		return err
	}
	return nil
}

func (sp *Spudo) loadConfig(configPath string) error {
	// Set default config
	sp.Config = getDefaultConfig()

	if _, err := toml.DecodeFile(configPath, &sp.Config); err != nil {
		return errors.New("Failed to read config - " + err.Error())
	}

	if sp.Config.Token == "" {
		return errors.New("No token in config")
	}
	return nil
}

// Start will add handler functions to the Session and open the
// websocket connection
func (sp *Spudo) Start() {
	rand.Seed(time.Now().UnixNano())

	var err error
	if sp.SpudoSession, err = newSpudoSession(sp.Config.Token, sp.logger); err != nil {
		sp.logger.fatal(err.Error())
	}

	if sp.Config.AudioEnabled {
		sp.addAudioCommands()
		sp.audioSessions = make(map[string]*spAudio)
		go sp.watchForDisconnect()
		sp.logger.info("Audio commands added")
	}

	if sp.Config.RESTEnabled {
		go sp.startRESTApi()
	}

	sp.AddHandler(sp.onReady)
	sp.AddHandler(sp.onMessageCreate)

	if err := sp.Open(); err != nil {
		sp.logger.fatal("Error opening websocket connection -", err)
	}

	sp.logger.info("Bot is now running. Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	sp.quit()
}

// quit handles everything that needs to occur for the bot to shutdown cleanly.
func (sp *Spudo) quit() {
	sp.logger.info("Bot is now shutting down")
	for _, as := range sp.audioSessions {
		if err := as.Voice.Disconnect(); err != nil {
			sp.logger.fatal("Error disconnecting from voice channel:", err)
		}
	}
	if err := sp.Close(); err != nil {
		sp.logger.fatal("Error closing discord session:", err)
	}
	os.Exit(1)
}

func (sp *Spudo) onReady(s *discordgo.Session, r *discordgo.Ready) {
	for _, p := range sp.startupPlugins {
		p.Exec()
	}

	if !sp.TimersStarted {
		sp.startTimedMessages()
	}
}

func (sp *Spudo) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Always ignore bot users (including itself)
	if m.Author.Bot {
		return
	}

	go sp.handleCommand(m)
	go sp.handleUserReaction(m)
	go sp.handleMessageReaction(m)
}

// sendPrivateMessage creates a UserChannel before attempting to send
// a message directly to a user rather than in the server channel.
func (sp *Spudo) sendPrivateMessage(userID string, message interface{}) {
	privChannel, err := sp.UserChannelCreate(userID)
	if err != nil {
		sp.logger.error("Error creating private channel -", err)
		return
	}
	switch v := message.(type) {
	case string:
		sp.SendMessage(privChannel.ID, v)
	case *discordgo.MessageEmbed:
		sp.SendEmbed(privChannel.ID, v)
	}
}

// respondToUser is a helper method around SendMessage that will
// mention the user who created the message.
func (sp *Spudo) respondToUser(m *discordgo.MessageCreate, response string) {
	sp.SendMessage(m.ChannelID, m.Author.Mention()+" "+response)
}

// attemptCommand will check if comStr is in the commands map. If it
// is, it will return the command response as resp and whether or not
// the message should be sent privately as private.
func (sp *Spudo) attemptCommand(author, channel, comStr string, args []string) (resp interface{}, private bool) {
	if com, isValid := sp.spudoCommands[comStr]; isValid {
		resp = com.Exec(author, channel, args...)
		return
	}

	if com, isValid := sp.commands[comStr]; isValid {
		resp = com.Exec(author, args)
		private = com.PrivateResponse
		return
	}
	return unknownCommand(sp.Config.UnknownCommandMessage), private
}

func (sp *Spudo) handleCommand(m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, sp.Config.CommandPrefix) {
		return
	}
	if !sp.canPost(m.Author.ID) {
		sp.respondToUser(m, sp.Config.CooldownMessage)
		return
	}

	commandText := strings.Split(strings.TrimPrefix(m.Content, sp.Config.CommandPrefix), " ")

	com := strings.ToLower(commandText[0])
	args := commandText[1:]

	commandResp, isPrivate := sp.attemptCommand(m.Author.ID, m.ChannelID, com, args)

	switch v := commandResp.(type) {
	case nil: // For commands that do not need a response
	case string:
		if isPrivate {
			sp.sendPrivateMessage(m.Author.ID, v)
		} else {
			sp.respondToUser(m, v)
		}
		sp.startCooldown(m.Author.ID)
	case *Embed:
		if isPrivate {
			sp.sendPrivateMessage(m.Author.ID, v.MessageEmbed)
		} else {
			sp.SendEmbed(m.ChannelID, v.MessageEmbed)
		}
		sp.startCooldown(m.Author.ID)
	case *Complex:
		defer v.file.Close()
		sp.SendComplex(m.ChannelID, v.MessageSend)
		sp.startCooldown(m.Author.ID)
	case voiceCommand:
		sp.SendMessage(m.ChannelID, string(v))
	case unknownCommand:
		sp.respondToUser(m, string(v))
	}
}

func (sp *Spudo) handleUserReaction(m *discordgo.MessageCreate) {
	for _, ur := range sp.userReactions {
		for _, user := range ur.UserIDs {
			if user == m.Author.ID {
				for _, reaction := range ur.ReactionIDs {
					sp.AddReaction(m, reaction)
				}
			}
		}
	}
}

func (sp *Spudo) handleMessageReaction(m *discordgo.MessageCreate) {
	for _, mr := range sp.messageReactions {
		for _, trigger := range mr.TriggerWords {
			if strings.Contains(strings.ToLower(m.Content), strings.ToLower(trigger)) {
				for _, reaction := range mr.ReactionIDs {
					sp.AddReaction(m, reaction)
				}
			}
		}
	}
}

// Returns whether or not the user can issue a command based on a timer.
func (sp *Spudo) canPost(user string) bool {
	if userTime, isValid := sp.CooldownList[user]; isValid {
		return time.Since(userTime).Seconds() > float64(sp.Config.CooldownTimer)
	}
	return true
}

// Adds user to cooldown list.
func (sp *Spudo) startCooldown(user string) {
	sp.CooldownList[user] = time.Now()
}

// Starts all TimedMessages.
func (sp *Spudo) startTimedMessages() {
	for _, p := range sp.timedMessages {

		c := cron.New(cron.WithLocation(time.UTC))

		if _, err := c.AddFunc(p.CronString, func() {
			timerFunc := p.Exec()
			switch v := timerFunc.(type) {
			case string:
				for _, chanID := range p.Channels {
					sp.SendMessage(chanID, v)
				}
			case *Embed:
				for _, chanID := range p.Channels {
					sp.SendEmbed(chanID, v.MessageEmbed)

				}
			}
		}); err != nil {
			sp.logger.error("Error starting "+p.Name+" timed message - ", err)
			continue
		}
		c.Start()
	}

	sp.TimersStarted = true
}
