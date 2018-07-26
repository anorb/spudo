package main

import "github.com/anorb/spudo/pluginhandler"

func ping(args []string) interface{} {
	return "Pong!"
}

func beep(args []string) interface{} {
	return "Boop!"
}

// Register ...
func Register() interface{} {
	var commands []*pluginhandler.CommandPlugin

	commands = append(commands, pluginhandler.NewCommand("ping", ping).SetDescription("Responds to !ping with 'Pong!'"))

	commands = append(commands, pluginhandler.NewCommand("beep", beep).SetDescription("Responds to !beep with 'Boop!'"))

	return commands
}
