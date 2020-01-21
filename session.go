package spudo

import (
	"github.com/bwmarrin/discordgo"
)

type SpudoSession struct {
	*discordgo.Session
	logger *spudoLogger
}

func newSpudoSession(token string, logger *spudoLogger) (*SpudoSession, error) {
	ss := &SpudoSession{}
	var err error
	ss.logger = logger
	ss.Session, err = discordgo.New("Bot " + token)
	return ss, err
}

// SendMessage is a helper function around ChannelMessageSend from
// discordgo. It will send a message to a given channel.
func (ss *SpudoSession) SendMessage(channelID string, message string) {
	_, err := ss.ChannelMessageSend(channelID, message)
	if err != nil {
		ss.logger.info("Failed to send message response -", err)
	}
}
