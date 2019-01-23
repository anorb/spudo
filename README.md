# Spudo
Spudo helps build a Discord bot that is easily extensible to add functionality. Built using [DiscordGo](https://github.com/bwmarrin/discordgo).

## Basic usage
### Create a config
The config uses the [TOML](https://github.com/toml-lang/toml) format. If you don't make one, you will be prompted to and the default settings will be used.
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
# Enable audio capability
AudioEnabled=true
# Enable REST capability and which port it listens on
RESTEnabled=true
RESTPort="8889"
```
### Create bot
```go
package main

import (
	"fmt"

	"github.com/anorb/spudo"
)

func main() {
	bot := spudo.NewSpudo()
	bot.AddCommand("ping", "responds with pong", ping)
	bot.Start()
}

func ping(args []string) interface{} {
	return "Pong!"
}
```
Further examples can be found [here](./examples/bot/main.go).

## Advanced features
### Audio
Spudo has audio playback which only works with Youtube (for now). Simply set AudioEnabled to true in the config and the commands will be enabled.

- !play *youtube-link* will cause the bot to join your channel and begin playing the audio from the link
  - If there is already audio playing, it will queue the audio and play it next
- !pause will pause currently playing audio
- !skip will skip the current track and play the next one if there is one available

### REST API
Spudo has the ability to start a REST API that can be used to send messages to a specific channel when it receives a hit. Point any webhooks you want at it and parse the request. Be sure to enable this feature in the config and choose a port to listen on.
```go
var (
	bot           = spudo.NewSpudo()
	sendChannelID = "channel-id-here"
)

func main() {
	bot.AddRESTRoute("endpoint", eventHandler)
	bot.Start()
}

func eventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
            bot.SendMessage(sendChannelID, "Received something!")
	}
}
```
## FAQ

### What kind of plugins can be made?

- Return a string response to a command
- Return a Discord embed message to a command
- Add reactions to specific user's messages
- Add reactions to a message containing specific strings
- Send a message (string or embed) at specific time
  - The second argument in this example uses [cron-style](https://en.wikipedia.org/wiki/Cron) strings to define when the messages should be sent

All of these are demonstrated [here](./examples/bot/main.go).

### What if I want to put the config.toml file somewhere else?
```sh
./bot -config=./path/to/wherever
```
