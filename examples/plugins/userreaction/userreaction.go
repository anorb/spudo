package main

import "gitlab.com/anorb/plugo/pluginhandler"

// Register ...
func Register() interface{} {
	return pluginhandler.NewUserReaction("userreaction", "138198523144306689", "👍")
}
