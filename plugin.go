package spudo

type commandPlugin struct {
	Name            string                          // Name of command
	Exec            func(args []string) interface{} // Function that will be executed when command is used
	Description     string                          // Description of command for a help command to use
	PrivateResponse bool                            // Indicates whether or not the command will yield a private message response
}

type timedMessagePlugin struct {
	Name       string             // Name of the timed message
	CronString string             // Cron-style string to determine when the Exec function is executed
	Exec       func() interface{} // Function that will be executed based on the CronString
}

type userReactionPlugin struct {
	Name        string   // Name of plugin
	UserIDs     []string // UserIDs of users the reactions will apply to
	ReactionIDs []string // ReactionIDs of the reactions that will be applied to the message
}

type messageReactionPlugin struct {
	Name         string   // Name of plugin
	TriggerWords []string // Words that will trigger the reactions being adde
	ReactionIDs  []string // Reactions that will be added when triggered
}
