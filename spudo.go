package spudo

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"plugin"
	"strings"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bwmarrin/discordgo"
	"github.com/robfig/cron"
	"gitlab.com/anorb/spudo/embed"
	"gitlab.com/anorb/spudo/pluginhandler"
	"gitlab.com/anorb/spudo/utils"
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
	PluginsDir            string
}

// Bot contains everything about the bot itself
type Bot struct {
	Session                *discordgo.Session
	Config                 Config
	CommandPlugins         map[string]*pluginhandler.CommandPlugin
	TimedMessagePlugins    []*pluginhandler.TimedMessagePlugin
	UserReactionPlugins    []*pluginhandler.UserReactionPlugin
	MessageReactionPlugins []*pluginhandler.MessageReactionPlugin
	CooldownList           map[string]time.Time
	TimersStarted          bool
}

// NewBot will create a new Bot and return it. It will also load
// Config and all plugins that are currently available.
func NewBot() *Bot {
	bot := &Bot{}
	bot.CooldownList = make(map[string]time.Time)
	bot.CommandPlugins = make(map[string]*pluginhandler.CommandPlugin)
	bot.TimedMessagePlugins = make([]*pluginhandler.TimedMessagePlugin, 0)
	bot.UserReactionPlugins = make([]*pluginhandler.UserReactionPlugin, 0)
	bot.MessageReactionPlugins = make([]*pluginhandler.MessageReactionPlugin, 0)
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
		PluginsDir:            "plugins",
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
	b.Config.Token, err = getInput("Eneter token: ")
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
		return fmt.Errorf("Error reading config - %v", err.Error())
	}

	if b.Config.Token == "" {
		return fmt.Errorf("Error: No Token set in config")
	}

	if b.Config.DefaultChannelID == "" {
		log.Println("WARNING: No DefaultChannelID set in config")
		log.Println("Welcome back message and timed messages will not be sent")
	}

	return nil
}

func (b *Bot) createSession() error {
	session, err := discordgo.New("Bot " + b.Config.Token)
	if err != nil {
		return fmt.Errorf("Error creating Discord session - %v", err.Error())
	}
	b.Session = session
	return nil
}

// loadPlugins gets all the plugins (any .so file in the pluginsDir)
// and looks for the Register function then executes it. It then adds
// the plugin to the applicable map or slice.
func (b *Bot) loadPlugins() error {
	if _, err := os.Stat(b.Config.PluginsDir); os.IsNotExist(err) {
		return fmt.Errorf("Error reading the plugin directory - %v", err)
	}

	pluginFiles, err := filepath.Glob(fmt.Sprintf("%s/*.so", b.Config.PluginsDir))
	if err != nil {
		return fmt.Errorf("Globbing pattern malformed - %v", err)
	}

	for _, filename := range pluginFiles {
		p, err := plugin.Open(filename)
		if err != nil {
			fmt.Printf("Error opening plugin, ignoring: %s - %s\n", filename, err)
			continue
		}

		reg, err := p.Lookup("Register")
		if err != nil {
			fmt.Println("Register function not found in plugin, ignoring: " + filename)
			continue
		}

		registerPlugin, ok := reg.(func() interface{})
		if !ok {
			fmt.Println("Error with plugin Register function, ignoring: " + filename)
			continue
		}
		c := registerPlugin()

		switch v := c.(type) {
		case *pluginhandler.CommandPlugin:
			b.addCommandPlugin(v)
		case *pluginhandler.TimedMessagePlugin:
			b.addTimedMessagePlugin(v)
		case *pluginhandler.UserReactionPlugin:
			b.addUserReactionPlugin(v)
		case *pluginhandler.MessageReactionPlugin:
			b.addMessageReactionPlugin(v)
		case []*pluginhandler.CommandPlugin:
			for _, p := range v {
				b.addCommandPlugin(p)
			}
		case []*pluginhandler.TimedMessagePlugin:
			for _, p := range v {
				b.addTimedMessagePlugin(p)
			}
		case []*pluginhandler.UserReactionPlugin:
			for _, p := range v {
				b.addUserReactionPlugin(p)
			}
		case []*pluginhandler.MessageReactionPlugin:
			for _, p := range v {
				b.addMessageReactionPlugin(p)
			}
		default:
			fmt.Printf("Failed to load plugin: %s - Unknown plugin type\n", filename)
		}
	}
	return nil
}

func (b *Bot) addCommandPlugin(plugin *pluginhandler.CommandPlugin) {
	b.CommandPlugins[strings.ToLower(plugin.Name)] = plugin
	fmt.Printf("%s plugin registered as a command\n", plugin.Name)
}

func (b *Bot) addTimedMessagePlugin(plugin *pluginhandler.TimedMessagePlugin) {
	b.TimedMessagePlugins = append(b.TimedMessagePlugins, plugin)
	fmt.Printf("%s plugin registered as a timed message\n", plugin.Name)
}

func (b *Bot) addUserReactionPlugin(plugin *pluginhandler.UserReactionPlugin) {
	b.UserReactionPlugins = append(b.UserReactionPlugins, plugin)
	fmt.Printf("%s plugin registered as a user reaction\n", plugin.Name)
}

func (b *Bot) addMessageReactionPlugin(plugin *pluginhandler.MessageReactionPlugin) {
	b.MessageReactionPlugins = append(b.MessageReactionPlugins, plugin)
	fmt.Printf("%s plugin registered as a message reaction\n", plugin.Name)
}

// Start will add handler function to the Session and open the
// websocket connection.
func (b *Bot) Start() {
	configPath := flag.String("config", "./config.toml", "TODO")
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	// Check if config exists, if it doesn't use
	// createMinimalConfig to generate one.
	if _, err := os.Stat(*configPath); os.IsNotExist(err) {
		fmt.Println("Config not detected, attempting to create...")
		if err := b.createMinimalConfig(); err != nil {
			log.Fatal(err)
		}
	}

	if err := b.loadConfig(*configPath); err != nil {
		log.Fatal(err)
	}

	if err := b.createSession(); err != nil {
		log.Fatal(err)
	}

	if err := b.loadPlugins(); err != nil {
		log.Fatal(err)
	}

	b.Session.AddHandler(b.onReady)
	b.Session.AddHandler(b.onMessageCreate)

	if err := b.Session.Open(); err != nil {
		log.Fatal("Error opening websocket connection - " + err.Error())
	}

	fmt.Println("Bot is now running. Press CTRL-C to exit.")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-c

	fmt.Println("Bot is now shutting down.")

	if err := b.Session.Close(); err != nil {
		log.Fatal("Error closing discord session" + err.Error())
	}
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
		log.Println("Failed to send message response - " + err.Error())
	}
}

// sendEmbed is a helper function around ChannelMessageSendEmbed from
// discordgo. It will send an embed message to a given channel.
func (b *Bot) sendEmbed(channelID string, embed *discordgo.MessageEmbed) {
	_, err := b.Session.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		log.Println("Failed to send embed message response - " + err.Error())
	}
}

// sendPrivateMessage creates a UserChannel before attempting to send
// a message directly to a user rather than in the server channel.
func (b *Bot) sendPrivateMessage(userID string, message interface{}) {
	privChannel, err := b.Session.UserChannelCreate(userID)
	if err != nil {
		log.Println("Error creating private channel - " + err.Error())
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
		log.Println("Error adding reaction - " + err.Error())
	}
}

// attemptCommand will check if comStr is in the CommandPlugins
// map. If it is, it will return the command response as resp and
// whether or not the message should be sent privately as private.
func (b *Bot) attemptCommand(comStr string, args []string) (resp interface{}, private bool) {
	if com, isValid := b.CommandPlugins[comStr]; isValid {
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
	case *utils.Embed:
		e := embed.Convert(v)

		if isPrivate {
			b.sendPrivateMessage(m.Author.ID, e.MessageEmbed)
		} else {
			b.sendEmbed(m.ChannelID, e.MessageEmbed)
		}
		b.startCooldown(m.Author.ID)
	default:
		b.respondToUser(m, b.Config.UnknownCommandMessage)
	}
}

func (b *Bot) handleUserReaction(m *discordgo.MessageCreate) {
	for _, plugin := range b.UserReactionPlugins {
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
	for _, plugin := range b.MessageReactionPlugins {
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
	for _, p := range b.TimedMessagePlugins {

		c := cron.NewWithLocation(time.UTC)
		timerFunc := p.Exec()

		if err := c.AddFunc(p.CronString, func() {
			switch v := timerFunc.(type) {
			case string:
				b.sendMessage(b.Config.DefaultChannelID, v)
			case *utils.Embed:
				e := embed.Convert(v)
				b.sendEmbed(b.Config.DefaultChannelID, e.MessageEmbed)
			}
		}); err != nil {
			log.Println("Error starting time message - " + p.Name + ": " + err.Error())
			continue
		}
		c.Start()
	}

	b.TimersStarted = true
}
