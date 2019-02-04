package spudo

type command struct {
	Name            string                          // Name of the command
	Exec            func(args []string) interface{} // Function that will be executed when command is used
	Description     string                          // Description of command for a help command to use
	PrivateResponse bool                            // Indicates whether or not the command will yield a private message response
}

type startupPlugin struct {
	Name string
	Exec func()
}

type timedMessage struct {
	Name       string             // Name of the timed message
	Channel    string             // ID of channel the message should be sent in
	CronString string             // Cron-style string to determine when the Exec function is executed
	Exec       func() interface{} // Function that will be executed
}

type userReaction struct {
	Name        string   // Name of the user reaction
	UserIDs     []string // UserIDs of users the reactions will apply to
	ReactionIDs []string // ReactionIDs of the reactions that will be applied to the message
}

type messageReaction struct {
	Name         string   // Name of the message reaction
	TriggerWords []string // Words that will trigger the reactions being added
	ReactionIDs  []string // Reactions that will be added when triggered
}

type spudoCommand struct {
	Name            string                                                   // Name of the command
	Exec            func(author, channel string, args ...string) interface{} // Function that will be executed when command is used
	Description     string                                                   // Description of command for a help command to use
	PrivateResponse bool                                                     // Indicates whether or not the command will yield a private message response
}
