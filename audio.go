package spudo

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"sync"
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

var (
	errBadVoiceState    = errors.New("Unable to find user voice state")
	errBadQueuePosition = errors.New("Bad queue position")
	errEndOfQueue       = errors.New("End of queue")
)

var vcSameChannelMsg = voiceCommand("you must be in the same channel to use this command")

type voiceCommand string

type ytAudio struct {
	*ytdl.VideoInfo
	dlURL       *url.URL
	sendChannel string
}

type audioQueue struct {
	sync.Mutex
	playlist []*ytAudio
	position int
}

func newAudioQueue() *audioQueue {
	return &audioQueue{
		playlist: []*ytAudio{},
		position: 0,
	}
}

func (q *audioQueue) add(audio *ytAudio) int {
	q.Lock()
	defer q.Unlock()

	q.playlist = append(q.playlist, audio)
	return len(q.playlist) - q.position
}

func (q *audioQueue) next() error {
	q.Lock()
	defer q.Unlock()

	if q.position+1 >= 0 && q.position+1 < len(q.playlist) {
		q.position++
		return nil
	}
	return errEndOfQueue
}

func (q *audioQueue) current() (*ytAudio, error) {
	q.Lock()
	defer q.Unlock()

	if q.position+1 >= 0 && q.position < len(q.playlist) {
		return q.playlist[q.position], nil
	}
	return nil, errBadQueuePosition
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
	go sp.startAudio()
	return nil
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
	audioPos := sp.audioQueue.add(a)

	return voiceCommand("queued `" + a.VideoInfo.Title + "` in position " + strconv.Itoa(audioPos))
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

func (sp *Spudo) startAudio() {
	err := sp.Voice.Speaking(true)
	if err != nil {
		sp.logger.error("Failed setting speaking: ", err)
		return
	}
	defer sp.Voice.Speaking(false)

	sp.audioStatus = audioPlay

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	for {
		audio, err := sp.audioQueue.current()
		if err != nil {
			sp.logger.error("Error getting current song in queue: ", err)
			break
		}

		encodingSession, err := dca.EncodeFile(audio.dlURL.String(), options)
		if err != nil {
			sp.logger.error("Error encoding file: ", err)
			break
		}
		defer encodingSession.Cleanup()
		done := make(chan error)
		stream := dca.NewStream(encodingSession, sp.Voice, done)

		nowPlaying := fmt.Sprintf("now playing: `%v`", audio.Title)
		duration := fmt.Sprintf("duration: `%v`", audio.Duration)
		ch := audio.sendChannel
		sp.sendMessage(ch, nowPlaying+"\n"+duration)

		err = sp.sendAudio(stream, done)
		if err != nil {
			sp.logger.error("Error sending audio: ", err)
		}

		// If the stop command is issued, sendAudio would
		// return and we break out of the audio loop here
		if sp.audioStatus == audioStop {
			break
		}

		// If this returns non-nil, we know we've reached the
		// end of the queue
		err = sp.audioQueue.next()
		if err != nil {
			break
		}
	}
}

func (sp *Spudo) sendAudio(stream *dca.StreamingSession, done chan error) error {
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
				return nil
			case audioStop:
				sp.audioStatus = audioStop
				return nil
			}
		case err := <-done:
			if err != nil {
				if err != io.ErrUnexpectedEOF && err != io.EOF {
					return err
				}
			}
			return nil
		}
	}
}
