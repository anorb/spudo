package spudo

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron"
)

// Config contains all options for the config file
type Config struct {
	Token                 string
	CommandPrefix         string
	DefaultChannelID      string
	CooldownTimer         int
	WelcomeBackMessage    string
	CooldownMessage       string
	UnknownCommandMessage string
	AudioEnabled          bool
}

// Spudo contains everything about the bot itself
type Spudo struct {
	Session       *discordgo.Session
	Config        Config
	CooldownList  map[string]time.Time
	TimersStarted bool
	logger        *spudoLogger

	spudoCommands map[string]*spudoCommand

	commands         map[string]*command
	timedMessages    []*timedMessage
	userReactions    []*userReaction
	messageReactions []*messageReaction

	Voice        *discordgo.VoiceConnection
	audioQueue   []*ytAudio
	audioControl chan int
	audioStatus  int
}

// NewSpudo will initialize everything Spudo needs to run.
func NewSpudo() *Spudo {
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

// Ask user for input using prompt and returns the entry and any
// errors
func getInput(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.Replace(input, "\n", "", -1), nil
}

// Returns the default config settings for the bot
func getDefaultConfig() Config {
	return Config{
		CommandPrefix:         "!",
		CooldownTimer:         10,
		CooldownMessage:       "Too many commands at once!",
		WelcomeBackMessage:    "I'm back!",
		UnknownCommandMessage: "Invalid command!",
	}
}

// createMinimalConfig prompts the user to enter a Token and
// DefaultChannelID for the Config. This is used if no config is
// found.
func (sp *Spudo) createMinimalConfig() error {
	// Set default config
	sp.Config = getDefaultConfig()
	path := "./config.toml"

	var err error
	sp.Config.Token, err = getInput("Enter token: ")
	if err != nil {
		return err
	}

	sp.Config.DefaultChannelID, err = getInput("Enter default channel ID: ")
	if err != nil {
		return err
	}

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

	if sp.Config.DefaultChannelID == "" {
		sp.logger.info("No DefaultChannelID set in config - Welcome back message and timed messages will not be sent")
	}

	return nil
}

func (sp *Spudo) createSession() error {
	session, err := discordgo.New("Bot " + sp.Config.Token)
	if err != nil {
		return errors.New("Error creating Discord session - " + err.Error())
	}
	sp.Session = session
	return nil
}

// Start will add handler functions to the Session and open the
// websocket connection
func (sp *Spudo) Start() {
	configPath := flag.String("config", "./config.toml", "TODO")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	// Check if config exists, if it doesn't use
	// createMinimalConfig to generate one.
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		sp.logger.info("Config not detected, attempting to create...")
		if err := sp.createMinimalConfig(); err != nil {
			sp.logger.fatal("Failed to create minimal config", err)
		}
	}

	if err := sp.loadConfig(*configPath); err != nil {
		sp.logger.fatal(err.Error())
	}

	if err := sp.createSession(); err != nil {
		sp.logger.fatal(err.Error())
	}

	if sp.Config.AudioEnabled {
		sp.addAudioCommands()
		sp.audioControl = make(chan int)
		sp.audioStatus = audioStop
		go sp.watchForDisconnect()
		sp.logger.info("Audio commands added")
	}

	sp.Session.AddHandler(sp.onReady)
	sp.Session.AddHandler(sp.onMessageCreate)

	if err := sp.Session.Open(); err != nil {
		sp.logger.fatal("Error opening websocket connection -", err)
	}

	sp.logger.info("Bot is now running. Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-c

	sp.quit()
}

// quit handles everything that needs to occur for the bot to shutdown cleanly.
func (sp *Spudo) quit() {
	sp.logger.info("Bot is now shutting down")
	if sp.Voice != nil {
		if err := sp.Voice.Disconnect(); err != nil {
			sp.logger.fatal("Error disconnecting from voice channel:", err)
		}
	}
	if err := sp.Session.Close(); err != nil {
		sp.logger.fatal("Error closing discord session:", err)
	}
	os.Exit(1)
}

func (sp *Spudo) onReady(s *discordgo.Session, r *discordgo.Ready) {
	if sp.Config.WelcomeBackMessage != "" && sp.Config.DefaultChannelID != "" {
		sp.sendMessage(sp.Config.DefaultChannelID, sp.Config.WelcomeBackMessage)
	}
	if !sp.TimersStarted && sp.Config.DefaultChannelID != "" {
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

// sendMessage is a helper function around ChannelMessageSend from
// discordgo. It will send a message to a given channel.
func (sp *Spudo) sendMessage(channelID string, message string) {
	_, err := sp.Session.ChannelMessageSend(channelID, message)
	if err != nil {
		sp.logger.error("Failed to send message response -", err)
	}
}

// sendEmbed is a helper function around ChannelMessageSendEmbed from
// discordgo. It will send an embed message to a given channel.
func (sp *Spudo) sendEmbed(channelID string, embed *discordgo.MessageEmbed) {
	_, err := sp.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		sp.logger.error("Failed to send embed message response -", err)
	}
}

// sendPrivateMessage creates a UserChannel before attempting to send
// a message directly to a user rather than in the server channel.
func (sp *Spudo) sendPrivateMessage(userID string, message interface{}) {
	privChannel, err := sp.Session.UserChannelCreate(userID)
	if err != nil {
		sp.logger.error("Error creating private channel -", err)
		return
	}
	switch v := message.(type) {
	case string:
		sp.sendMessage(privChannel.ID, v)
	case *discordgo.MessageEmbed:
		sp.sendEmbed(privChannel.ID, v)
	}
}

// respondToUser is a helper method around sendMessage that will
// mention the user who created the message.
func (sp *Spudo) respondToUser(m *discordgo.MessageCreate, response string) {
	sp.sendMessage(m.ChannelID, fmt.Sprintf("%s", m.Author.Mention()+" "+response))
}

// addReaction is a helper method around MessageReactionAdd from
// discordgo. It adds a reaction to a given message.
func (sp *Spudo) addReaction(m *discordgo.MessageCreate, reactionID string) {
	if err := sp.Session.MessageReactionAdd(m.ChannelID, m.ID, reactionID); err != nil {
		sp.logger.error("Error adding reaction -", err)
	}
}

// attemptCommand will check if comStr is in the commands map. If it
// is, it will return the command response as resp and whether or not
// the message should be sent privately as private.
func (sp *Spudo) attemptCommand(author, channel, comStr string, args []string) (resp interface{}, private bool) {
	if com, isValid := sp.spudoCommands[comStr]; isValid {
		resp = com.Exec(author, channel, args...)
		private = com.PrivateResponse
		return
	}

	if com, isValid := sp.commands[comStr]; isValid {
		resp = com.Exec(args)
		private = com.PrivateResponse
		return
	}
	return
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
	args := commandText[1:len(commandText)]

	commandResp, isPrivate := sp.attemptCommand(m.Author.ID, m.ChannelID, com, args)

	switch v := commandResp.(type) {
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
			sp.sendEmbed(m.ChannelID, v.MessageEmbed)
		}
		sp.startCooldown(m.Author.ID)
	case voiceCommand:
		sp.sendMessage(m.ChannelID, string(v))
	default:
		sp.respondToUser(m, sp.Config.UnknownCommandMessage)
	}
}

func (sp *Spudo) handleUserReaction(m *discordgo.MessageCreate) {
	for _, ur := range sp.userReactions {
		for _, user := range ur.UserIDs {
			if user == m.Author.ID {
				for _, reaction := range ur.ReactionIDs {
					sp.addReaction(m, reaction)
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
					sp.addReaction(m, reaction)
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

		c := cron.NewWithLocation(time.UTC)

		if err := c.AddFunc(p.CronString, func() {
			timerFunc := p.Exec()
			switch v := timerFunc.(type) {
			case string:
				sp.sendMessage(sp.Config.DefaultChannelID, v)
			case *Embed:
				sp.sendEmbed(sp.Config.DefaultChannelID, v.MessageEmbed)
			}
		}); err != nil {
			sp.logger.error("Error starting "+p.Name+" timed message - ", err)
			continue
		}
		c.Start()
	}

	sp.TimersStarted = true
}
