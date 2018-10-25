package main

import (
	"github.com/anorb/spudo"
	_ "github.com/anorb/spudo/examples/plugins/catgif"
	_ "github.com/anorb/spudo/examples/plugins/embed"
	_ "github.com/anorb/spudo/examples/plugins/fiveseconds"
	_ "github.com/anorb/spudo/examples/plugins/hello"
	_ "github.com/anorb/spudo/examples/plugins/messagereaction"
	_ "github.com/anorb/spudo/examples/plugins/ping"
	_ "github.com/anorb/spudo/examples/plugins/userreaction"
)

func main() {
	bot := spudo.NewBot()
	bot.Start()
}
