package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/anorb/plugo"
)

func main() {
	bot := plugo.NewBot()
	bot.Start()

	fmt.Println("Bot is now running. Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	if err := bot.Session.Close(); err != nil {
		log.Fatal("Error closing discord session" + err.Error())
	}
}
