package main

import "gitlab.com/anorb/plugo/pluginhandler"

// Register ...
func Register() interface{} {
	userIDs := []string{"351684465131886746541"}
	reactions := []string{"👍", "☝"}
	return pluginhandler.NewUserReaction("userreaction", userIDs, reactions)
}
