package main

import "github.com/anorb/spudo/pluginhandler"

func ping(args []string) interface{} {
	return "Pong!"
}

// Register ...
func Register() interface{} {
	return pluginhandler.NewCommand("ping", ping).SetDescription("Responds to !ping with pong")
}
