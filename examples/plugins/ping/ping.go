package main

import "gitlab.com/anorb/plugo/pluginhandler"

func ping(args []string) interface{} {
	return "Pong!"
}

// Register ...
func Register() interface{} {
	return pluginhandler.NewCommand("ping", ping).SetDescription("Responds to !ping with pong")
}