package spudo

import (
	"github.com/bwmarrin/discordgo"
)

type session struct {
	*discordgo.Session
	logger *spudoLogger
}

func newSession(token string, logger *spudoLogger) (*session, error) {
	ss := &session{}
	var err error
	ss.logger = logger
	ss.Session, err = discordgo.New("Bot " + token)
	return ss, err
}

// SendMessage is a helper function around ChannelMessageSend from
// discordgo. It will send a message to a given channel.
func (ss *session) SendMessage(channelID string, message string) {
	_, err := ss.ChannelMessageSend(channelID, message)
	if err != nil {
		ss.logger.info("Failed to send message response -", err)
	}
}

// SendEmbed is a helper function around ChannelMessageSendEmbed from
// discordgo. It will send an embed message to a given channel.
func (ss *session) SendEmbed(channelID string, embed *discordgo.MessageEmbed) {
	_, err := ss.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		ss.logger.error("Failed to send embed message response -", err)
	}
}

func (ss *session) SendComplex(channelID string, ms *discordgo.MessageSend) {
	_, err := ss.ChannelMessageSendComplex(channelID, ms)
	if err != nil {
		ss.logger.error("Failed to send complex message response -", err)
	}
}

// AddReaction is a helper method around MessageReactionAdd from
// discordgo. It adds a reaction to a given message.
func (ss *session) AddReaction(m *discordgo.MessageCreate, reactionID string) {
	if err := ss.MessageReactionAdd(m.ChannelID, m.ID, reactionID); err != nil {
		ss.logger.error("Error adding reaction -", err)
	}
}
