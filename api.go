package spudo

import (
	"net/http"
)

// AddCommand will add a command that will trigger Exec.
func (sp *Spudo) AddCommand(name, description string, exec func(author string, args []string) interface{}) {
	if _, ok := sp.commands[name]; ok {
		sp.logger.info("Failed to add command: ", name, "- Already exists")
		return
	}
	sp.commands[name] = &command{
		Name:        name,
		Description: description,
		Exec:        exec,
	}
	sp.logger.info("Command added: ", name)
}

// AddStartupPlugin will trigger exec when the bot initially starts
// up.
func (sp *Spudo) AddStartupPlugin(name string, exec func()) {
	p := &startupPlugin{
		Name: name,
		Exec: exec,
	}
	sp.startupPlugins = append(sp.startupPlugins, p)
	sp.logger.info("Startup plugin added: ", name)
}

// AddTimedMessage will trigger Exec at specific times to send a
// message.
func (sp *Spudo) AddTimedMessage(name, cronString string, channels []string, exec func() interface{}) {
	p := &timedMessage{
		Name:       name,
		Channels:   channels,
		CronString: cronString,
		Exec:       exec,
	}
	sp.timedMessages = append(sp.timedMessages, p)
	sp.logger.info("Timed message added: ", name)
}

// AddUserReaction will add reaction(s) to a user(s) message.
func (sp *Spudo) AddUserReaction(name string, userIDs, reactionIDs []string) {
	p := &userReaction{
		Name:        name,
		UserIDs:     userIDs,
		ReactionIDs: reactionIDs,
	}
	sp.userReactions = append(sp.userReactions, p)
	sp.logger.info("User reaction added: ", name)
}

// AddMessageReaction will add reaction(s) when trigger word(s) are in a
// message.
func (sp *Spudo) AddMessageReaction(name string, triggerWords, reactionIDs []string) {
	p := &messageReaction{
		Name:         name,
		TriggerWords: triggerWords,
		ReactionIDs:  reactionIDs,
	}
	sp.messageReactions = append(sp.messageReactions, p)
	sp.logger.info("Message reaction added: ", name)
}

// AddRESTRoute will add an endpoint at route that will execute exec
// when used.
func (sp *Spudo) AddRESTRoute(route string, exec func(w http.ResponseWriter, r *http.Request)) {
	if !sp.Config.RESTEnabled {
		sp.logger.info("Failed to add REST route - REST API is disabled")
		return
	}
	http.HandleFunc("/"+route, exec)
	sp.logger.info("REST route added: ", route)
}
