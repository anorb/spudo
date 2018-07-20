package main

import "gitlab.com/anorb/plugo/pluginhandler"

// Register ...
func Register() interface{} {
	triggerWords := []string{"ok", "okay"}
	reactions := []string{"👌"}
	return pluginhandler.NewMessageReaction("messagereaction", triggerWords, reactions)
}
