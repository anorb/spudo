package main

import "gitlab.com/anorb/plugo/pluginhandler"

// Register ...
func Register() interface{} {
	return pluginhandler.NewAddReaction("userreaction", "478431893247966985", "👍")
}
