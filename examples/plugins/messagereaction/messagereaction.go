package main

import "github.com/anorb/spudo/pluginhandler"

// Register ...
func Register() interface{} {
	triggerWords := []string{"ok", "okay"}
	reactions := []string{"👌"}
	return pluginhandler.NewMessageReaction("messagereaction", triggerWords, reactions)
}
