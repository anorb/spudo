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

func NewComplex(message, fileName string) *Complex {
	c := &Complex{}
	var err error
	c.file, err = os.Open(fileName)
	if err != nil {
		log.Println("Error opening file: ", err)
	}

	c.MessageSend = &discordgo.MessageSend{
		Content: message,
		Files: []*discordgo.File{
			{
				Name:   fileName,
				Reader: c.file,
			},
		},
	}
	return c
}

func NewComplexAttachment(fileName string) *Complex {
	c := &Complex{}
	var err error
	c.file, err = os.Open(fileName)
	if err != nil {
		log.Println("Error opening file: ", err)
	}

	c.MessageSend = &discordgo.MessageSend{
		Files: []*discordgo.File{
			{
				Name:   fileName,
				Reader: c.file,
			},
		},
	}
	return c
}
