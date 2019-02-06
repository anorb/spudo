package main

import (
	"fmt"

	"github.com/anorb/spudo"
)

func main() {
	bot := spudo.NewSpudo()
	bot.AddCommand("embed", "test embed command", embed)
	bot.AddCommand("hello", "says hello + whatever argument follows", hello)
	bot.AddCommand("ping", "responds with pong", ping)

	bot.AddStartupPlugin("welcome message", func() {
		bot.SendMessage("789654132546789", "I'm back!")
	})

	bot.AddTimedMessage("five seconds", "0,5,10,15,20,25,30,35,40,45,50,55 * * * * *", []string{"354846132188644643"}, timer)

	bot.AddMessageReaction("reacts to ok", []string{"ok"}, []string{"👌"})

	bot.AddUserReaction("userreaction", []string{"56418947165489476"}, []string{"👌"})

	bot.Start()
}

func embed(args []string) interface{} {
	return spudo.NewEmbed().SetTitle("This is a test").SetDescription("This is also a test")
}

func hello(args []string) interface{} {
	return fmt.Sprintf("Hello %s", args[0])
}

func ping(args []string) interface{} {
	return "Pong!"
}

func timer() interface{} {
	return "Five seconds have elapsed"
}
