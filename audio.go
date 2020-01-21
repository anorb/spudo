package spudo

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/rylio/ytdl"
)

const (
	statusPlay = iota
	statusSkip
	statusPause
	statusResume
	statusStop
)

var (
	errBadVoiceState    = errors.New("Unable to find user voice state")
	errBadQueuePosition = errors.New("Bad queue position")
	errEndOfQueue       = errors.New("End of queue")
)

var vcSameChannelMsg = voiceCommand("you must be in the same channel to use this command")

type voiceCommand string

type media struct {
	*ytdl.VideoInfo
	dlURL       *url.URL
	sendChannel string
}

type spAudio struct {
	sync.Mutex
	Voice   *discordgo.VoiceConnection
	control chan int
	queue   *mediaQueue
	status  int
}

func newSpAudio() *spAudio {
	return &spAudio{
		control: make(chan int),
		status:  statusStop,
		queue:   newMediaQueue(),
	}
}

type mediaQueue struct {
	sync.Mutex
	playlist []*media
	position int
}

func newMediaQueue() *mediaQueue {
	return &mediaQueue{
		playlist: []*media{},
		position: 0,
	}
}

func (q *mediaQueue) add(m *media) int {
	q.Lock()
	defer q.Unlock()

	q.playlist = append(q.playlist, m)
	return len(q.playlist) - q.position
}

func (q *mediaQueue) next() error {
	q.Lock()
	defer q.Unlock()

	q.position++
	if q.position >= 0 && q.position < len(q.playlist) {
		return nil
	}
	return errEndOfQueue
}

func (q *mediaQueue) current() (*media, error) {
	q.Lock()
	defer q.Unlock()

	if q.position+1 >= 0 && q.position < len(q.playlist) {
		return q.playlist[q.position], nil
	}
	return nil, errBadQueuePosition
}

func (sp *Spudo) addAudioCommands() {
	// Explicitly set audio commands outside of the standard
	// AddCommand method so audio commands will overwrite anything
	// with the same name
	sp.spudoCommands["play"] = &spudoCommand{
		Name:        "play",
		Description: "add media to playlist",
		Exec:        sp.cmdPlayMedia,
	}
	sp.spudoCommands["pause"] = &spudoCommand{
		Name:        "pause",
		Description: "pause/unpause currently playing media",
		Exec:        sp.cmdPauseMedia,
	}
	sp.spudoCommands["skip"] = &spudoCommand{
		Name:        "skip",
		Description: "skip currently playing media",
		Exec:        sp.cmdSkipMedia,
	}
}

func (sp *Spudo) watchForDisconnect() {
	for range time.Tick(1 * time.Second) {
		for _, as := range sp.audioSessions {
			userCount, err := sp.getListenerCount(as.Voice.GuildID, as.Voice.ChannelID)
			if err != nil {
				sp.logger.error("Error getting listener count: ", err)
				continue
			}
			if userCount <= 1 {
				if as.status == statusPlay || as.status == statusPause {
					as.control <- statusStop
				}

				err := as.Voice.Disconnect()
				if err != nil {
					sp.logger.error("Error disconnecting from voice channel:", err)
				}
				sp.removeAudioSession(as.Voice.GuildID)
			}
		}
	}
}

// Returns the number of users in the same voice channel
func (sp *Spudo) getListenerCount(guildid, channelid string) (count int, err error) {
	g, err := sp.Session.Guild(guildid)
	if err != nil {
		return
	}
	for _, vs := range g.VoiceStates {
		if channelid == vs.ChannelID {
			count++
		}
	}
	return
}

func (sp *Spudo) getAudioSession(id string) (*spAudio, bool) {
	sp.Lock()
	defer sp.Unlock()
	audioSess, exists := sp.audioSessions[id]
	if !exists {
		return nil, false
	}
	return audioSess, true
}

func (sp *Spudo) addAudioSession(author string) (*spAudio, error) {
	sp.Lock()
	defer sp.Unlock()
	audioSess := newSpAudio()

	var err error

	audioSess.Voice, err = sp.joinUserVoiceChannel(author)
	if err != nil {
		return nil, err
	}

	sp.audioSessions[audioSess.Voice.GuildID] = audioSess

	return audioSess, nil
}

func (sp *Spudo) removeAudioSession(id string) {
	sp.Lock()
	defer sp.Unlock()
	delete(sp.audioSessions, id)
}

func (sp *Spudo) cmdPlayMedia(author, channel string, args ...string) interface{} {
	sp.CommandMutex.Lock()
	defer sp.CommandMutex.Unlock()
	if len(args) < 1 {
		return voiceCommand("play requires a link argument")
	}

	vs, err := sp.getUserVoiceState(author)
	if err != nil {
		sp.logger.error("Error getting voice state: ", err)
		return voiceCommand("err")
	}

	audioSess, exists := sp.getAudioSession(vs.GuildID)
	if !exists {
		var err error

		audioSess, err = sp.addAudioSession(author)
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

	// If media is actively being played, return the queued message from queueMedia
	if audioSess.status == statusPlay || audioSess.status == statusPause {
		return audioSess.queueMedia(args[0], channel)
	}

	audioSess.queueMedia(args[0], channel)
	audioSess.status = statusPlay
	go audioSess.start(sp.SpudoSession)
	return nil
}

func (sp *Spudo) cmdPauseMedia(author, channel string, args ...string) interface{} {
	vs, err := sp.getUserVoiceState(author)
	if err != nil {
		sp.logger.error("Error getting voice state: ", err)
		return voiceCommand("err")
	}

	audioSess, exists := sp.getAudioSession(vs.GuildID)
	if !exists {
		return voiceCommand("unable to pause, not in channel")
	}

	if !sp.userInVoiceChannel(author) {
		return vcSameChannelMsg
	}

	var vc voiceCommand
	if audioSess.status == statusPause {
		audioSess.control <- statusResume
		vc = voiceCommand("resuming audio")
	} else if audioSess.status == statusPlay {
		audioSess.control <- statusPause
		vc = voiceCommand("pausing audio")
	}
	return vc
}

func (sp *Spudo) cmdSkipMedia(author, channel string, args ...string) interface{} {
	vs, err := sp.getUserVoiceState(author)
	if err != nil {
		sp.logger.error("Error getting voice state: ", err)
		return voiceCommand("err")
	}

	audioSess, exists := sp.getAudioSession(vs.GuildID)
	if !exists {
		return voiceCommand("unable to pause, not in channel")
	}

	if !sp.userInVoiceChannel(author) {
		return vcSameChannelMsg
	}

	if audioSess.status != statusPlay && audioSess.status != statusPause {
		return voiceCommand("can't skip, no audio playing")
	}
	audioSess.control <- statusSkip
	return voiceCommand("skipping...")
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
	if err != nil {
		sp.logger.error("Error finding user voice state: ", err)
		return false
	}
	sp.Lock()
	defer sp.Unlock()
	for _, as := range sp.audioSessions {
		if vc.ChannelID == as.Voice.ChannelID {
			return true
		}
	}
	return false
}

func (sa *spAudio) queueMedia(audioLink, channel string) voiceCommand {
	a := new(media)
	var err error
	a.VideoInfo, err = ytdl.GetVideoInfo(audioLink)
	if err != nil {
		log.Println("Error getting video info: ", err)
		return voiceCommand("failed to add item to queue")
	}

	format := a.VideoInfo.Formats.Extremes(ytdl.FormatAudioBitrateKey, true)[0]
	a.dlURL, err = a.VideoInfo.GetDownloadURL(format)
	if err != nil {
		log.Println("Error getting download url: ", err)
		return voiceCommand("failed to add item to queue")
	}

	a.sendChannel = channel
	audioPos := sa.queue.add(a)

	return voiceCommand("queued `" + a.VideoInfo.Title + "` in position " + strconv.Itoa(audioPos))
}

func (sa *spAudio) start(sess *SpudoSession) {
	err := sa.Voice.Speaking(true)
	if err != nil {
		log.Println("Failed setting speaking: ", err)
		return
	}

	defer func() {
		err := sa.Voice.Speaking(false)
		log.Println("Failed to end speaking: ", err)
	}()

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	for {
		audio, err := sa.queue.current()
		if err != nil {
			log.Println("Error getting current song in queue: ", err)
			break
		}

		encodingSession, err := dca.EncodeFile(audio.dlURL.String(), options)
		if err != nil {
			log.Println("Error encoding file: ", err)
			break
		}
		defer encodingSession.Cleanup()
		done := make(chan error)
		stream := dca.NewStream(encodingSession, sa.Voice, done)

		nowPlaying := fmt.Sprintf("now playing: `%v`", audio.Title)
		duration := fmt.Sprintf("duration: `%v`", audio.Duration)
		sess.SendMessage(audio.sendChannel, nowPlaying+"\n"+duration)

		err = sa.send(stream, done)
		if err != nil {
			log.Println("Error sending audio: ", err)
		}

		// If the stop command is issued, the send method
		// would return and we break out of the audio loop
		// here
		if sa.status == statusStop {
			break
		}

		// If this returns non-nil, we know we've reached the
		// end of the queue
		err = sa.queue.next()
		if err != nil {
			sa.status = statusStop
			break
		}
	}
}

func (sa *spAudio) send(stream *dca.StreamingSession, done chan error) error {
	for {
		select {
		case cmd := <-sa.control:
			switch cmd {
			case statusPause:
				sa.status = statusPause
				stream.SetPaused(true)
			case statusResume:
				sa.status = statusPlay
				stream.SetPaused(false)
			case statusSkip:
				stream.SetPaused(true)
				return nil
			case statusStop:
				sa.status = statusStop
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
