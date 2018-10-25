package hello

import (
	"fmt"

	"github.com/anorb/spudo"
)

func hello(args []string) interface{} {
	return fmt.Sprintf("Hello %s", args[0])
}

func init() {
	spudo.AddCommandPlugin("hello", "says hello + whatever argument follows", hello)
}
