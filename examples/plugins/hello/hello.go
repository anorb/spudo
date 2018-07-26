package main

import (
	"fmt"

	"github.com/anorb/spudo/pluginhandler"
)

func hello(args []string) interface{} {
	return fmt.Sprintf("Hello %s", args[0])
}

func Register() interface{} {
	return pluginhandler.NewCommand("hello", hello)
}
