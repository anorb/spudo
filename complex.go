package spudo

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

type Complex struct {
	*discordgo.MessageSend
	file *os.File
}

func NewComplex(fileName string) *Complex {
	c := &Complex{}
	var err error
	c.file, err = os.Open(fileName)
	if err != nil {
		log.Println("Error opening file: ", err)
	}

	c.MessageSend = &discordgo.MessageSend{
		Files: []*discordgo.File{
			&discordgo.File{
				Name:   fileName,
				Reader: c.file,
			},
		},
	}
	return c
}
