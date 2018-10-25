package embed

import (
	"github.com/anorb/spudo"
)

func embed(args []string) interface{} {
	return spudo.NewEmbed().SetTitle("This is a test").SetDescription("This is also a test")
}

func init() {
	spudo.AddCommandPlugin("embed", "test embed command", embed)
}
