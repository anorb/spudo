package plugo

import (
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"plugin"
	"strings"
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
	AddReactionPlugin   []*pluginhandler.AddReactionPlugin
	CooldownList        map[string]time.Time
	TimersStarted       bool
}

func loadConfig() Config {
	c := Config{
		CommandPrefix:         "!",
		CooldownTimer:         10,
		CooldownMessage:       "Too many commands at once!",
		WelcomeBackMessage:    "I'm back!",
		UnknownCommandMessage: "Invalid command!",
		PluginsDir:            "plugins",
	}

	if _, err := toml.DecodeFile("config.toml", &c); err != nil {
		log.Fatal("Error reading config - " + err.Error())
	}

	if c.Token == "" {
		log.Fatal("Error: No Token set in config")
	}

	if c.DefaultChannelID == "" {
		log.Println("WARNING: No DefaultChannelID set in config")
		log.Println("Welcome back message and timed messages will not be sent")
	}

	return c
}

// loadPlugins gets all the plugins (any .so file) and looks for the
// Register function then executes it. It then adds it to the
// applicable map or slice and returns the various types of plugins.
func loadPlugins(pluginsDir string) (map[string]*pluginhandler.CommandPlugin, []*pluginhandler.TimedMessagePlugin, []*pluginhandler.AddReactionPlugin) {
	pluginFiles, err := filepath.Glob(fmt.Sprintf("%s/*.so", pluginsDir))
	if err != nil {
		log.Panic("Error read the plugin directory")
	}

	var commandPlugins = make(map[string]*pluginhandler.CommandPlugin)
	var timedMessagePlugins = make([]*pluginhandler.TimedMessagePlugin, 0)
	var addReactionPlugins = make([]*pluginhandler.AddReactionPlugin, 0)

	for _, filename := range pluginFiles {
		p, err := plugin.Open(filename)
		if err != nil {
			log.Fatalf("Error opening plugin: %s - %s", filename, err)
		}

		reg, err := p.Lookup("Register")
		if err != nil {
			log.Println("Register function not found in plugin: " + filename)
			continue
		}

		registerPlugin, ok := reg.(func() interface{})
		if !ok {
			log.Println("Error registering plugin: " + filename)
			continue
		}
		c := registerPlugin()

		switch v := c.(type) {
		case *pluginhandler.CommandPlugin:
			commandPlugins[strings.ToLower(v.Name)] = v
			fmt.Printf("%s plugin registered as a command\n", v.Name)
		case *pluginhandler.TimedMessagePlugin:
			timedMessagePlugins = append(timedMessagePlugins, v)
			fmt.Printf("%s plugin registered as a timed message\n", v.Name)
		case *pluginhandler.AddReactionPlugin:
			addReactionPlugins = append(addReactionPlugins, v)
			fmt.Printf("%s plugin registered as an add reaction\n", v.Name)
		case []*pluginhandler.CommandPlugin:
			for _, p := range v {
				commandPlugins[strings.ToLower(p.Name)] = p
				fmt.Printf("%s plugin registered as a command\n", p.Name)
			}
		default:
			fmt.Printf("Failed to load plugin: %s - Unknown plugin type\n", filename)
		}
	}
	return commandPlugins, timedMessagePlugins, addReactionPlugins
}

// NewBot will create a new Bot and return it. It will also load
// Config and all plugins that are currently available.
func NewBot() *Bot {
	rand.Seed(time.Now().UnixNano())

	bot := &Bot{}
	bot.Config = loadConfig()

	session, err := discordgo.New("Bot " + bot.Config.Token)
	if err != nil {
		log.Fatal("Error creating Discord session - " + err.Error())
	}
	bot.Session = session

	bot.CooldownList = make(map[string]time.Time)

	bot.CommandPlugins, bot.TimedMessagePlugins, bot.AddReactionPlugin = loadPlugins(bot.Config.PluginsDir)

	return bot
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
	for _, reaction := range b.AddReactionPlugin {
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
