package main

import (
	"gitlab.com/anorb/plugo/pluginhandler"
)

func timer() interface{} {
	return "Five seconds have elapsed"
}

// Register ...
func Register() interface{} {
	return pluginhandler.NewTimedMessage("five seconds", "0,5,10,15,20,25,30,35,40,45,50,55 * * * * *", timer)
}
