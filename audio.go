package spudo

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/rylio/ytdl"
)

const (
	audioPlay = iota
	audioSkip
	audioPause
	audioResume
)

type voiceCommand string

type ytAudio struct {
	*ytdl.VideoInfo
	dlURL       *url.URL
	sendChannel string
}

func (sp *Spudo) addAudioCommands() {
	sp.spudoCommands["join"] = &spudoCommand{
		Name:        "join",
		Description: "join voice",
		Exec:        sp.joinVoice,
	}
	sp.spudoCommands["leave"] = &spudoCommand{
		Name:        "leave",
		Description: "leave voice",
		Exec:        sp.leaveVoice,
	}
	sp.spudoCommands["play"] = &spudoCommand{
		Name:        "play",
		Description: "play next in queue",
		Exec:        sp.playAudio,
	}
	sp.spudoCommands["pause"] = &spudoCommand{
		Name:        "pause",
		Description: "pause audio",
		Exec:        sp.pauseAudio,
	}
	sp.spudoCommands["resume"] = &spudoCommand{
		Name:        "resume",
		Description: "resume audio",
		Exec:        sp.resumeAudio,
	}
	sp.spudoCommands["skip"] = &spudoCommand{
		Name:        "skip",
		Description: "skip current track",
		Exec:        sp.skipAudio,
	}
}

func (sp *Spudo) joinVoice(author, channel string, args ...string) interface{} {
	var err error

	if sp.Voice != nil {
		fmt.Println(sp.Voice.ChannelID)
		return voiceCommand("already in voice channel")
	}

	sp.Voice, err = sp.joinUserVoiceChannel(author)
	if err != nil {
		sp.logger.error("Error joining voice: ", err)
		return voiceCommand("error joining voice channel")
	}
	return voiceCommand("joined voice channel")
}

func (sp *Spudo) leaveVoice(author, channel string, args ...string) interface{} {
	if sp.Voice == nil {
		return voiceCommand("can't leave, not connected")
	}

	sp.audioPlaying = false

	err := sp.Voice.Disconnect()
	if err != nil {
		sp.logger.error("Error disconnecting from voice channel:", err)
		return voiceCommand("error leaving voice")
	}
	sp.Voice = nil
	return voiceCommand("left voice chat")
}

func (sp *Spudo) playAudio(author, channel string, args ...string) interface{} {
	if sp.Voice == nil {
		return voiceCommand("unable to play, not in channel")
	}

	if len(args) < 1 {
		return voiceCommand("play requires a link argument")
	}

	if sp.audioPlaying {
		return sp.queueAudio(args[0], channel)
	}

	sp.queueAudio(args[0], channel)
	pn, _ := sp.playNext()
	return voiceCommand(pn)
}

func (sp *Spudo) pauseAudio(author, channel string, args ...string) interface{} {
	if !sp.audioPlaying {
		return voiceCommand("can't pause, no audio playing")
	}
	sp.audioControl <- audioPause
	return voiceCommand("pausing audio")
}

func (sp *Spudo) resumeAudio(author, channel string, args ...string) interface{} {
	if !sp.audioPlaying {
		return voiceCommand("can't resume, no audio playing")
	}
	sp.audioControl <- audioResume
	return voiceCommand("resuming audio")
}

func (sp *Spudo) skipAudio(author, channel string, args ...string) interface{} {
	if !sp.audioPlaying {
		return voiceCommand("can't skip, no audio playing")
	}
	sp.audioControl <- audioSkip
	return voiceCommand("skipping current audio")
}

func (sp *Spudo) playNext() (string, string) {
	nowPlaying := "now playing `" + sp.audioQueue[0].Title + "`"
	ch := sp.audioQueue[0].sendChannel
	go sp.startAudio(sp.audioQueue[0])
	sp.audioQueue = append(sp.audioQueue[:0], sp.audioQueue[0+1:]...)
	return nowPlaying, ch
}

func (sp *Spudo) queueAudio(audioLink, channel string) voiceCommand {
	a := new(ytAudio)
	var err error
	a.VideoInfo, err = ytdl.GetVideoInfo(audioLink)
	if err != nil {
		sp.logger.error("Error getting video info: ", err)
		return voiceCommand("failed to add item to queue")
	}

	format := a.VideoInfo.Formats.Extremes(ytdl.FormatAudioBitrateKey, true)[0]
	a.dlURL, err = a.VideoInfo.GetDownloadURL(format)
	if err != nil {
		sp.logger.error("Error getting download url: ", err)
		return voiceCommand("failed to add item to queue")
	}

	a.sendChannel = channel
	sp.audioQueue = append(sp.audioQueue, a)

	return voiceCommand("queued `" + a.VideoInfo.Title + "` in position " + strconv.Itoa(len(sp.audioQueue)))
}

func (sp *Spudo) getUserVoiceState(userid string) (*discordgo.VoiceState, error) {
	for _, guild := range sp.Session.State.Guilds {
		for _, vs := range guild.VoiceStates {
			if vs.UserID == userid {
				return vs, nil
			}
		}
	}
	return nil, errors.New("Unable to find user voice state")
}

func (sp *Spudo) joinUserVoiceChannel(userID string) (*discordgo.VoiceConnection, error) {
	vs, err := sp.getUserVoiceState(userID)
	if err != nil {
		return nil, err
	}
	return sp.Session.ChannelVoiceJoin(vs.GuildID, vs.ChannelID, false, true)
}

func (sp *Spudo) startAudio(a *ytAudio) {
	err := sp.Voice.Speaking(true)
	if err != nil {
		sp.logger.error("Failed setting speaking: ", err)
		return
	}

	sp.audioPlaying = true

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	encodingSession, err := dca.EncodeFile(a.dlURL.String(), options)
	if err != nil {
		sp.logger.error("Error encoding file: ", err)
		return
	}
	defer encodingSession.Cleanup()

	done := make(chan error)
	stream := dca.NewStream(encodingSession, sp.Voice, done)

AudioLoop:
	for {
		select {
		case cmd := <-sp.audioControl:
			switch cmd {
			case audioPause:
				stream.SetPaused(true)
			case audioResume:
				stream.SetPaused(false)
			case audioSkip:
				stream.SetPaused(true)
				if len(sp.audioQueue) > 0 {
					pn, ch := sp.playNext()
					sp.sendMessage(ch, pn)
				} else {
					sp.audioPlaying = false
					err = sp.Voice.Speaking(false)
					if err != nil {
						sp.logger.error("Failed to end speaking: ", err)
					}
				}
				break AudioLoop
			}
		case err := <-done:
			if err != nil {
				if err != io.ErrUnexpectedEOF && err != io.EOF {
					sp.logger.error("Audio error: ", err)
				}
			}

			encodingSession.Truncate()

			if len(sp.audioQueue) > 0 {
				pn, ch := sp.playNext()
				sp.sendMessage(ch, pn)
			} else {
				sp.audioPlaying = false
				err = sp.Voice.Speaking(false)
				if err != nil {
					sp.logger.error("Failed to end speaking: ", err)
				}
			}
			break AudioLoop
		}
	}
}
