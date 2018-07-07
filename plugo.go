package plugo

import (
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
	"gitlab.com/anorb/plugo/embed"
	"gitlab.com/anorb/plugo/pluginhandler"
	"gitlab.com/anorb/plugo/utils"
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
	Session             *discordgo.Session
	Config              Config
	CommandPlugins      map[string]*pluginhandler.CommandPlugin
	TimedMessagePlugins []*pluginhandler.TimedMessagePlugin
	AddReactionPlugins  []*pluginhandler.AddReactionPlugin
	CooldownList        map[string]time.Time
	TimersStarted       bool
}

// NewBot will create a new Bot and return it. It will also load
// Config and all plugins that are currently available.
func NewBot() *Bot {
	rand.Seed(time.Now().UnixNano())

	bot := &Bot{}

	if err := bot.loadConfig(); err != nil {
		log.Fatal(err)
	}

	session, err := discordgo.New("Bot " + bot.Config.Token)
	if err != nil {
		log.Fatal("Error creating Discord session - " + err.Error())
	}
	bot.Session = session

	bot.CooldownList = make(map[string]time.Time)

	if err := bot.loadPlugins(); err != nil {
		log.Fatal(err)
	}

	return bot
}

func (b *Bot) loadConfig() error {
	// Set default config
	b.Config = Config{
		CommandPrefix:         "!",
		CooldownTimer:         10,
		CooldownMessage:       "Too many commands at once!",
		WelcomeBackMessage:    "I'm back!",
		UnknownCommandMessage: "Invalid command!",
		PluginsDir:            "plugins",
	}

	if _, err := toml.DecodeFile("config.toml", &b.Config); err != nil {
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

// loadPlugins gets all the plugins (any .so file in the pluginsDir)
// and looks for the Register function then executes it. It then adds
// the plugin to the applicable map or slice.
func (b *Bot) loadPlugins() error {
	pluginFiles, err := filepath.Glob(fmt.Sprintf("%s/*.so", b.Config.PluginsDir))
	if err != nil {
		return fmt.Errorf("Error reading the plugin directory - pluginsDir value: %v", b.Config.PluginsDir)
	}

	b.CommandPlugins = make(map[string]*pluginhandler.CommandPlugin)
	b.TimedMessagePlugins = make([]*pluginhandler.TimedMessagePlugin, 0)
	b.AddReactionPlugins = make([]*pluginhandler.AddReactionPlugin, 0)

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
			b.CommandPlugins[strings.ToLower(v.Name)] = v
			fmt.Printf("%s plugin registered as a command\n", v.Name)
		case *pluginhandler.TimedMessagePlugin:
			b.TimedMessagePlugins = append(b.TimedMessagePlugins, v)
			fmt.Printf("%s plugin registered as a timed message\n", v.Name)
		case *pluginhandler.AddReactionPlugin:
			b.AddReactionPlugins = append(b.AddReactionPlugins, v)
			fmt.Printf("%s plugin registered as an add reaction\n", v.Name)
		case []*pluginhandler.CommandPlugin:
			for _, p := range v {
				b.CommandPlugins[strings.ToLower(p.Name)] = p
				fmt.Printf("%s plugin registered as a command\n", p.Name)
			}
		case []*pluginhandler.TimedMessagePlugin:
			for _, p := range v {
				b.TimedMessagePlugins = append(b.TimedMessagePlugins, p)
				fmt.Printf("%s plugin registered as a timed message\n", p.Name)
			}
		case []*pluginhandler.AddReactionPlugin:
			for _, p := range v {
				b.AddReactionPlugins = append(b.AddReactionPlugins, p)
				fmt.Printf("%s plugin registered as an add reaction\n", p.Name)
			}
		default:
			fmt.Printf("Failed to load plugin: %s - Unknown plugin type\n", filename)
		}
	}
	return nil
}

// Start will add handler function to the Session and open the
// websocket connection.
func (b *Bot) Start() {
	b.Session.AddHandler(b.onReady)
	b.Session.AddHandler(b.onMessageCreate)

	err := b.Session.Open()
	if err != nil {
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

	b.handleCommand(m)
	b.handleAddReaction(m)
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

func (b *Bot) handleAddReaction(m *discordgo.MessageCreate) {
	for _, reaction := range b.AddReactionPlugins {
		if reaction.UserID == m.Author.ID {
			b.addReaction(m, reaction.ReactionID)
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
