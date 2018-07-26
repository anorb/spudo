package main

import (
	"gitlab.com/anorb/spudo/pluginhandler"
	"gitlab.com/anorb/spudo/utils"
)

func embed(args []string) interface{} {
	return utils.NewEmbed().SetTitle("This is an embed").AddField("First", "field").AddField("Seconds", "field").SetAuthor("Author name", "https://i.imgur.com/Oa5DbkC.png").SetImage("https://i.imgur.com/Oa5DbkC.png")
}

// Register ...
func Register() interface{} {
	return pluginhandler.NewCommand("embed", embed)
}
