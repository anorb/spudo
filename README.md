# PluGo
PluGo helps build a Discord bot that utilizes [Go plugins](https://golang.org/pkg/plugin/) to add functionality. Built using [DiscordGo](https://github.com/bwmarrin/discordgo).

## Creating a bot
### Create a config
The config uses the [TOML](https://github.com/toml-lang/toml) format.
```toml
# Token generated by discord for your bot (Required)
Token="SAKkj343jnNajw429Je"
# Default channel for welcome back/timed messages to be sent in (Required)
DefaultChannelID="5432188465698448"

# Prefix used to determine when a command is issued (Optional, default: !)
CommandPrefix="$"
# Seconds that must elapse before a user can issue another command (Optional, default: 10)
CooldownTimer=5
# Message that will be sent when a user tries to issue too many commands in a short time (Optional, default: Too many commands at once!)
CooldownMessage="You have used too many commands"
# Message bot will send when it conects or reconnects (Optional, default: I'm back!)
WelcomeBackMessage="I'm back"
# Message that will be sent when a user issues an invalid command (Optional, default: Invalid command!)
UnknownCommandMessage="Command is invalid"
# Directory where the .so plugins will be stored (Optional, default: plugins)
PluginsDir="data/plugins"
```
### Create bot
```go
package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"gitlab.com/anorb/plugo"
)

func main() {
	bot := plugo.NewBot()
	bot.Start()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err := bot.Session.Close(); err != nil {
		log.Fatal("Error closing discord session" + err.Error())
	}
}
```
Examples can be found [here](https://gitlab.com/anorb/plugo/tree/master/examples/bot).

## Basic plugin
### Create the plugin
```go
package main

import "gitlab.com/anorb/plugo/pluginhandler"

func ping(args []string) interface{} {
	return "pong!"
}

func Register() interface{} {
	return pluginhandler.NewCommand("ping", ping).SetDescription("Returns ping on !pong command")
}
```
### Build the plugin
```sh
go build -buildmode=plugin ping.go
```
### Move ping.so to directory specified in your config

Examples of the different kinds of plugins can be found [here](https://gitlab.com/anorb/plugo/tree/master/examples/plugins) along with a [small script](https://gitlab.com/anorb/plugo/blob/master/examples/plugins/build.sh) to build plugins en masse.

## FAQ

### What kind of plugins can be made?

- [Return a string response to a command](https://gitlab.com/anorb/plugo/blob/master/examples/plugins/ping/ping.go)
- [Return a Discord embed message to a command](https://gitlab.com/anorb/plugo/blob/master/examples/plugins/embed/embed.go)
- [Add reaction to specific user's messages](https://gitlab.com/anorb/plugo/blob/master/examples/plugins/userreaction/userreaction.go)
- [Send a message (string or embed) at specific time](https://gitlab.com/anorb/plugo/blob/master/examples/plugins/fiveseconds/fiveseconds.go)

### Why are there two embed.go files (embed/embed.go) and (utils/embed.go)?

Due to the plugins being statically compiled, if I only had embed/embed.go it would require the plugin to be 10mb+ as it would pull in discordgo since Embed struct acts as a wrapper around discordgo.MessageEmbed. For the plugins, use utils/embed.go which has limited dependencies and allow PluGo to convert it inside the bot and keep the plugin .so files ~1mb.
