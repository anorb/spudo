package main

import "gitlab.com/anorb/plugo/pluginhandler"

// Register ...
func Register() interface{} {
	return pluginhandler.NewMessageReaction("messagereaction", "ok", "👌")
}
