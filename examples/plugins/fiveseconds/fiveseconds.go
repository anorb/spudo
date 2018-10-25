package fiveseconds

import "github.com/anorb/spudo"

func timer() interface{} {
	return "Five seconds have elapsed"
}

func init() {
	spudo.AddTimedMessagePlugin("five seconds", "0,5,10,15,20,25,30,35,40,45,50,55 * * * * *", timer)
}
