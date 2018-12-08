package spudo

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/rylio/ytdl"
)

const (
	audioPlay = iota
	audioSkip
	audioPause
	audioResume
	audioStop
)

var errBadVoiceState = errors.New("Unable to find user voice state")
var vcSameChannelMsg = voiceCommand("you must be in the same channel to use this command")

type voiceCommand string

type ytAudio struct {
	*ytdl.VideoInfo
	dlURL       *url.URL
	sendChannel string
}

func (sp *Spudo) addAudioCommands() {
	sp.spudoCommands["play"] = &spudoCommand{
		Name:        "play",
		Description: "play next in queue",
		Exec:        sp.playAudio,
	}
	sp.spudoCommands["pause"] = &spudoCommand{
		Name:        "pause",
		Description: "pause/unpause audio",
		Exec:        sp.togglePause,
	}
	sp.spudoCommands["skip"] = &spudoCommand{
		Name:        "skip",
		Description: "skip current track",
		Exec:        sp.skipAudio,
	}
}

func (sp *Spudo) watchForDisconnect() {
	for range time.Tick(1 * time.Second) {
		if sp.Voice == nil {
			continue
		}

		userCount := sp.getListenerCount()

		if userCount <= 1 {
			sp.audioControl <- audioStop

			err := sp.Voice.Disconnect()
			if err != nil {
				sp.logger.error("Error disconnecting from voice channel:", err)
			}
			sp.Voice = nil
		}
	}
}

// Returns the number of users in the same voice channel
func (sp *Spudo) getListenerCount() int {
	count := 0
	for _, guild := range sp.Session.State.Guilds {
		for _, vs := range guild.VoiceStates {
			if sp.Voice.ChannelID == vs.ChannelID {
				count++
			}
		}
	}
	return count
}

func (sp *Spudo) playAudio(author, channel string, args ...string) interface{} {
	if sp.Voice == nil {
		var err error
		sp.Voice, err = sp.joinUserVoiceChannel(author)
		if err != nil {
			if err == errBadVoiceState {
				return voiceCommand("you must be in a voice channel to use this command")
			}
			sp.logger.error("Error joining voice: ", err)
			return voiceCommand("error joining voice channel")
		}
	}

	if !sp.userInVoiceChannel(author) {
		return vcSameChannelMsg
	}

	if len(args) < 1 {
		return voiceCommand("play requires a link argument")
	}

	if sp.audioStatus == audioPlay || sp.audioStatus == audioPause {
		return sp.queueAudio(args[0], channel)
	}

	sp.queueAudio(args[0], channel)
	pn, _ := sp.playNext()
	return voiceCommand(pn)
}

func (sp *Spudo) togglePause(author, channel string, args ...string) interface{} {
	if sp.Voice == nil {
		return voiceCommand("unable to pause, not in channel")
	}

	if !sp.userInVoiceChannel(author) {
		return vcSameChannelMsg
	}

	var vc voiceCommand
	if sp.audioStatus == audioPause {
		sp.audioControl <- audioResume
		vc = voiceCommand("resuming audio")
	} else if sp.audioStatus == audioPlay {
		sp.audioControl <- audioPause
		vc = voiceCommand("pausing audio")
	}
	return vc
}

func (sp *Spudo) skipAudio(author, channel string, args ...string) interface{} {
	if sp.Voice == nil {
		return voiceCommand("unable to skip, not in channel")
	}

	if !sp.userInVoiceChannel(author) {
		return vcSameChannelMsg
	}

	if sp.audioStatus != audioPlay && sp.audioStatus != audioPause {
		return voiceCommand("can't skip, no audio playing")
	}
	sp.audioControl <- audioSkip
	return voiceCommand("skipping...")
}

func (sp *Spudo) playNext() (string, string) {
	nowPlaying := fmt.Sprintf("now playing: `%v`", sp.audioQueue[0].Title)
	duration := fmt.Sprintf("duration: `%v`", sp.audioQueue[0].Duration)
	ch := sp.audioQueue[0].sendChannel
	go sp.startAudio(sp.audioQueue[0])
	sp.audioQueue = append(sp.audioQueue[:0], sp.audioQueue[0+1:]...)
	return nowPlaying + "\n" + duration, ch
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
	return nil, errBadVoiceState
}

func (sp *Spudo) joinUserVoiceChannel(userID string) (*discordgo.VoiceConnection, error) {
	vs, err := sp.getUserVoiceState(userID)
	if err != nil {
		return nil, err
	}
	return sp.Session.ChannelVoiceJoin(vs.GuildID, vs.ChannelID, false, true)
}

func (sp *Spudo) userInVoiceChannel(userID string) bool {
	vc, err := sp.getUserVoiceState(userID)
	if err != nil && err != errBadVoiceState {
		sp.logger.error("Error finding user voice state: ", err)
		return false
	}
	if vc.ChannelID != sp.Voice.ChannelID {
		return false
	}
	return true
}

func (sp *Spudo) startAudio(a *ytAudio) {
	err := sp.Voice.Speaking(true)
	if err != nil {
		sp.logger.error("Failed setting speaking: ", err)
		return
	}

	sp.audioStatus = audioPlay

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
				sp.audioStatus = audioPause
				stream.SetPaused(true)
			case audioResume:
				sp.audioStatus = audioPlay
				stream.SetPaused(false)
			case audioSkip:
				stream.SetPaused(true)
				if len(sp.audioQueue) > 0 {
					pn, ch := sp.playNext()
					sp.sendMessage(ch, pn)
				} else {
					sp.audioStatus = audioStop
					err = sp.Voice.Speaking(false)
					if err != nil {
						sp.logger.error("Failed to end speaking: ", err)
					}
				}
				break AudioLoop
			case audioStop:
				sp.audioStatus = audioStop
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
				sp.audioStatus = audioStop
				err = sp.Voice.Speaking(false)
				if err != nil {
					sp.logger.error("Failed to end speaking: ", err)
				}
			}
			break AudioLoop
		}
	}
}
