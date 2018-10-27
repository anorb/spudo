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
}

// Bot contains everything about the bot itself
type Bot struct {
	Session       *discordgo.Session
	Config        Config
	CooldownList  map[string]time.Time
	TimersStarted bool
}

var (
	commandPlugins         = make(map[string]*commandPlugin)
	timedMessagePlugins    = make([]*timedMessagePlugin, 0)
	userReactionPlugins    = make([]*userReactionPlugin, 0)
	messageReactionPlugins = make([]*messageReactionPlugin, 0)
	logger                 = newLogger()
)

// AddCommandPlugin will add a regular command to the plugin map.
func AddCommandPlugin(command, description string, exec func(args []string) interface{}) {
	commandPlugins[command] = &commandPlugin{
		Name:        command,
		Description: description,
		Exec:        exec,
	}
	logger.info("Command plugin added:", command)
}

// AddTimedMessagePlugin will add a plugin that sends a message at
// specific times.
func AddTimedMessagePlugin(name, cronString string, exec func() interface{}) {
	p := &timedMessagePlugin{
		Name:       name,
		CronString: cronString,
		Exec:       exec,
	}
	timedMessagePlugins = append(timedMessagePlugins, p)
	logger.info("Timed message plugin added:", name)
}

// AddUserReactionPlugin will add a plugin that reacts to all user IDs
// with the reaction IDs.
func AddUserReactionPlugin(name string, userIDs, reactionIDs []string) {
	p := &userReactionPlugin{
		Name:        name,
		UserIDs:     userIDs,
		ReactionIDs: reactionIDs,
	}
	userReactionPlugins = append(userReactionPlugins, p)
	logger.info("User reaction plugin added:", name)
}

// AddMessageReactionPlugin will add a plugin that reacts to all
// trigger words with the reaction IDs.
func AddMessageReactionPlugin(name string, triggerWords, reactionIDs []string) {
	p := &messageReactionPlugin{
		Name:         name,
		TriggerWords: triggerWords,
		ReactionIDs:  reactionIDs,
	}
	messageReactionPlugins = append(messageReactionPlugins, p)
	logger.info("Message reaction plugin added:", name)
}

// NewBot will create a new Bot and return it. It will also load
// Config and all plugins that are currently available.
func NewBot() *Bot {
	bot := &Bot{}
	bot.CooldownList = make(map[string]time.Time)
	return bot
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
func (b *Bot) createMinimalConfig() error {
	// Set default config
	b.Config = getDefaultConfig()
	path := "./config.toml"

	var err error
	b.Config.Token, err = getInput("Enter token: ")
	if err != nil {
		return err
	}

	b.Config.DefaultChannelID, err = getInput("Enter default channel ID: ")
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	err = toml.NewEncoder(f).Encode(b.Config)
	if err != nil {
		return err
	}
	return nil
}

func (b *Bot) loadConfig(configPath string) error {
	// Set default config
	b.Config = getDefaultConfig()

	if _, err := toml.DecodeFile(configPath, &b.Config); err != nil {
		return errors.New("Failed to read config - " + err.Error())
	}

	if b.Config.Token == "" {
		return errors.New("No token in config")
	}

	if b.Config.DefaultChannelID == "" {
		logger.info("No DefaultChannelID set in config - Welcome back message and timed messages will not be sent")
	}

	return nil
}

func (b *Bot) createSession() error {
	session, err := discordgo.New("Bot " + b.Config.Token)
	if err != nil {
		return errors.New("Error creating Discord session - " + err.Error())
	}
	b.Session = session
	return nil
}

// Start will add handler functions to the Session and open the
// websocket connection
func (b *Bot) Start() {
	configPath := flag.String("config", "./config.toml", "TODO")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	// Check if config exists, if it doesn't use
	// createMinimalConfig to generate one.
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		logger.info("Config not detected, attempting to create...")
		if err := b.createMinimalConfig(); err != nil {
			logger.fatal("Failed to create minimal config", err)
		}
	}

	if err := b.loadConfig(*configPath); err != nil {
		logger.fatal(err.Error())
	}

	if err := b.createSession(); err != nil {
		logger.fatal(err.Error())
	}

	b.Session.AddHandler(b.onReady)
	b.Session.AddHandler(b.onMessageCreate)

	if err := b.Session.Open(); err != nil {
		logger.fatal("Error opening websocket connection -", err)
	}

	logger.info("Bot is now running. Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-c

	b.quit()
}

// quit handles everything that needs to occur for the bot to shutdown cleanly.
func (b *Bot) quit() {
	logger.info("Bot is now shutting down")
	if err := b.Session.Close(); err != nil {
		logger.fatal("Error closing discord session", err)
	}
	os.Exit(1)
}

func (b *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	if b.Config.WelcomeBackMessage != "" && b.Config.DefaultChannelID != "" {
		b.sendMessage(b.Config.DefaultChannelID, b.Config.WelcomeBackMessage)
	}
	if !b.TimersStarted && b.Config.DefaultChannelID != "" {
		b.startTimedMessages()
	}
}

func (b *Bot) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Always ignore bot users (including itself)
	if m.Author.Bot {
		return
	}

	go b.handleCommand(m)
	go b.handleUserReaction(m)
	go b.handleMessageReaction(m)
}

// sendMessage is a helper function around ChannelMessageSend from
// discordgo. It will send a message to a given channel.
func (b *Bot) sendMessage(channelID string, message string) {
	_, err := b.Session.ChannelMessageSend(channelID, message)
	if err != nil {
		logger.error("Failed to send message response -", err)
	}
}

// sendEmbed is a helper function around ChannelMessageSendEmbed from
// discordgo. It will send an embed message to a given channel.
func (b *Bot) sendEmbed(channelID string, embed *discordgo.MessageEmbed) {
	_, err := b.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		logger.error("Failed to send embed message response -", err)
	}
}

// sendPrivateMessage creates a UserChannel before attempting to send
// a message directly to a user rather than in the server channel.
func (b *Bot) sendPrivateMessage(userID string, message interface{}) {
	privChannel, err := b.Session.UserChannelCreate(userID)
	if err != nil {
		logger.error("Error creating private channel -", err)
		return
	}
	switch v := message.(type) {
	case string:
		b.sendMessage(privChannel.ID, v)
	case *discordgo.MessageEmbed:
		b.sendEmbed(privChannel.ID, v)
	}
}

// respondToUser is a helper method around sendMessage that will
// mention the user who created the message.
func (b *Bot) respondToUser(m *discordgo.MessageCreate, response string) {
	b.sendMessage(m.ChannelID, fmt.Sprintf("%s", m.Author.Mention()+" "+response))
}

// addReaction is a helper method around MessageReactionAdd from
// discordgo. It adds a reaction to a given message.
func (b *Bot) addReaction(m *discordgo.MessageCreate, reactionID string) {
	if err := b.Session.MessageReactionAdd(m.ChannelID, m.ID, reactionID); err != nil {
		logger.error("Error adding reaction -", err)
	}
}

// attemptCommand will check if comStr is in the CommandPlugins
// map. If it is, it will return the command response as resp and
// whether or not the message should be sent privately as private.
func (b *Bot) attemptCommand(comStr string, args []string) (resp interface{}, private bool) {
	if com, isValid := commandPlugins[comStr]; isValid {
		resp = com.Exec(args)
		private = com.PrivateResponse
		return
	}
	return
}

func (b *Bot) handleCommand(m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, b.Config.CommandPrefix) {
		return
	}
	if !b.canPost(m.Author.ID) {
		b.respondToUser(m, b.Config.CooldownMessage)
		return
	}

	commandText := strings.Split(strings.TrimPrefix(m.Content, b.Config.CommandPrefix), " ")

	for i, text := range commandText {
		commandText[i] = strings.ToLower(text)
	}

	com := commandText[0]
	args := commandText[1:len(commandText)]
	commandResp, isPrivate := b.attemptCommand(com, args)

	switch v := commandResp.(type) {
	case string:
		if isPrivate {
			b.sendPrivateMessage(m.Author.ID, v)
		} else {
			b.respondToUser(m, v)
		}
		b.startCooldown(m.Author.ID)
	case *Embed:
		if isPrivate {
			b.sendPrivateMessage(m.Author.ID, v.MessageEmbed)
		} else {
			b.sendEmbed(m.ChannelID, v.MessageEmbed)
		}
		b.startCooldown(m.Author.ID)
	default:
		b.respondToUser(m, b.Config.UnknownCommandMessage)
	}
}

func (b *Bot) handleUserReaction(m *discordgo.MessageCreate) {
	for _, plugin := range userReactionPlugins {
		for _, user := range plugin.UserIDs {
			if user == m.Author.ID {
				for _, reaction := range plugin.ReactionIDs {
					b.addReaction(m, reaction)
				}
			}
		}
	}
}

func (b *Bot) handleMessageReaction(m *discordgo.MessageCreate) {
	for _, plugin := range messageReactionPlugins {
		for _, trigger := range plugin.TriggerWords {
			if strings.Contains(strings.ToLower(m.Content), strings.ToLower(trigger)) {
				for _, reaction := range plugin.ReactionIDs {
					b.addReaction(m, reaction)
				}
			}
		}
	}
}

// Returns whether or not the user can issue a command based on a timer.
func (b *Bot) canPost(user string) bool {
	if userTime, isValid := b.CooldownList[user]; isValid {
		return time.Since(userTime).Seconds() > float64(b.Config.CooldownTimer)
	}
	return true
}

// Adds user to cooldown list.
func (b *Bot) startCooldown(user string) {
	b.CooldownList[user] = time.Now()
}

// Starts all TimedMessagePlugins.
func (b *Bot) startTimedMessages() {
	for _, p := range timedMessagePlugins {

		c := cron.NewWithLocation(time.UTC)

		if err := c.AddFunc(p.CronString, func() {
			timerFunc := p.Exec()
			switch v := timerFunc.(type) {
			case string:
				b.sendMessage(b.Config.DefaultChannelID, v)
			case *Embed:
				b.sendEmbed(b.Config.DefaultChannelID, v.MessageEmbed)
			}
		}); err != nil {
			logger.error("Error starting "+p.Name+" timed message - ", err)
			continue
		}
		c.Start()
	}

	b.TimersStarted = true
}
