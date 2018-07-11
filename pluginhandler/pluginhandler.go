package pluginhandler

// CommandPlugin contains information for a command
type CommandPlugin struct {
	Name            string                          // Name of command
	Exec            func(args []string) interface{} // Function that will be executed when command is used
	Description     string                          // Description of command for a help command to use
	PrivateResponse bool                            // Indicates whether or not the command will yield a private message response
}

// TimedMessagePlugin contains information for a timed message
type TimedMessagePlugin struct {
	Name       string             // Name of the timed message
	CronString string             // Cron-style string to determine when the Exec function is executed
	Exec       func() interface{} // Function that will be executed based on the CronString
}

// UserReactionPlugin contains information for a trigger to add reactions to a specific user's messages
type UserReactionPlugin struct {
	Name       string // Name of plugin
	UserID     string // UserID of user the reactions will apply to
	ReactionID string // ReactionID of the reaction that will be applied to the message
}

// MessageReactionPlugin contains information for a trigger to add
// reaction when a message contains a specific TriggerWord
type MessageReactionPlugin struct {
	Name        string
	TriggerWord string
	ReactionID  string
}

// NewCommand is a helper method that creates a new CommandPlugin with
// the required fields and returns the CommandPlugin
func NewCommand(name string, f func(args []string) interface{}) *CommandPlugin {
	return &CommandPlugin{Name: name, Exec: f}
}

// SetDescription is a helper method that adds the description to the
// Description field of the Plugin
func (p *CommandPlugin) SetDescription(description string) *CommandPlugin {
	p.Description = description
	return p
}

// SetPrivate is a helper method that sets the PrivateResponse field
// of Plugin to true indicating that the respond of the command will
// be sent via priate message
func (p *CommandPlugin) SetPrivate() *CommandPlugin {
	p.PrivateResponse = true
	return p
}

// NewTimedMessage is a helper method that creates a new
// TimedMessagePlugin with the required fields and returns the
// TimedMessagePlugin
func NewTimedMessage(name string, cronString string, f func() interface{}) *TimedMessagePlugin {
	return &TimedMessagePlugin{Name: name, CronString: cronString, Exec: f}
}

// NewUserReaction is a helper method that creates a new UserReactionPlugin
// with the required fields and returns the UserReactionPlugin
func NewUserReaction(name string, userID string, reactionID string) *UserReactionPlugin {
	return &UserReactionPlugin{Name: name, UserID: userID, ReactionID: reactionID}
}

// NewMessageReaction is a helper method that creates a
// MessageReactionPlugin with the required fields and returns the
// MessageReactionPlugin
func NewMessageReaction(name string, triggerWord string, reactionID string) *MessageReactionPlugin {
	return &MessageReactionPlugin{Name: name, TriggerWord: triggerWord, ReactionID: reactionID}
}
