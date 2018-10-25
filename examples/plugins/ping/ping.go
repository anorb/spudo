package ping

import (
	"github.com/anorb/spudo"
)

func ping(args []string) interface{} {
	return "Pong!"
}

func init() {
	spudo.AddCommandPlugin("ping", "responds with pong", ping)
}
