package spudo

import (
	"github.com/bwmarrin/discordgo"
)

const (
	embedLimitTitle       = 256
	embedLimitDescription = 2048
	embedLimitFieldValue  = 1024
	embedLimitFieldName   = 256
	embedLimitField       = 25
	embedLimitFooter      = 2048
	embedLimitAuthor      = 256
)

// Embed is a wrapper around *discordgo.MessageEmbed
type Embed struct {
	*discordgo.MessageEmbed
}

// NewEmbed returns a new Embed with no fields set
func NewEmbed() *Embed {
	return &Embed{&discordgo.MessageEmbed{}}
}

// SetTitle sets the Embed's Title to title. Will truncate title if it
// is too long. Returns the modified Embed.
func (e *Embed) SetTitle(title string) *Embed {
	if len(title) > embedLimitTitle {
		title = title[:embedLimitTitle]
	}

	e.Title = title
	return e
}

// SetDescription sets the Embed's Description to description. Will
// truncate description if it is too long. Returns the modified Embed.
func (e *Embed) SetDescription(description string) *Embed {
	if len(description) > embedLimitDescription {
		description = description[:embedLimitDescription]
	}

	e.Description = description
	return e
}

// AddField creates an EmbedField with name and value and adds it to
// to the Embed's Fields slice. If the embed has too many items in
// Fields, it will simply return the Embed as is. The value and name
// will be truncated if they are too long. Returns the modified Embed.
func (e *Embed) AddField(name, value string, inline bool) *Embed {
	if len(e.Fields) > embedLimitField {
		return e
	}
	if len(value) > embedLimitFieldName {
		value = value[:embedLimitFieldName]
	}
	if len(name) > embedLimitFieldValue {
		name = name[:embedLimitFieldValue]
	}

	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	})
	return e
}

// SetFooter creates an EmbedFooter and applies it to the Embed's
// Footer. If text is too long, it will be truncated. Returns the
// modified Embed.
// Parameters: text, iconURL, proxyURL
func (e *Embed) SetFooter(args ...string) *Embed {
	var text string
	var iconURL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		text = args[0]
		if len(text) > embedLimitFooter {
			text = text[:embedLimitFooter]
		}
	}
	if len(args) > 1 {
		iconURL = args[1]
	}
	if len(args) > 2 {
		proxyURL = args[2]
	}

	e.Footer = &discordgo.MessageEmbedFooter{
		IconURL:      iconURL,
		Text:         text,
		ProxyIconURL: proxyURL,
	}
	return e
}

// SetImage creates an EmbedImage and applies it to the Embed's
// image. Returns the modified Embed.
// Parameters: URL, proxyURL
func (e *Embed) SetImage(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}

	e.Image = &discordgo.MessageEmbedImage{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

// SetThumbnail creates an EmbedThumbnail and applies it to the
// Embed's Thumbnail field. Returns the modified Embed.
// Parameters: URL, proxyURL
func (e *Embed) SetThumbnail(args ...string) *Embed {
	var URL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		URL = args[0]
	}
	if len(args) > 1 {
		proxyURL = args[1]
	}

	e.Thumbnail = &discordgo.MessageEmbedThumbnail{
		URL:      URL,
		ProxyURL: proxyURL,
	}
	return e
}

// SetAuthor creates an EmbedAuthor and applies it to the Embed's
// Author field. Will truncate name if it's too long. Returns the
// modified Embed.
// Parameters: name, iconURL, URL, proxyURL
func (e *Embed) SetAuthor(args ...string) *Embed {
	var (
		name     string
		iconURL  string
		URL      string
		proxyURL string
	)

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		name = args[0]
		if len(name) > embedLimitAuthor {
			name = name[:embedLimitAuthor]
		}
	}
	if len(args) > 1 {
		iconURL = args[1]
	}
	if len(args) > 2 {
		URL = args[2]
	}
	if len(args) > 3 {
		proxyURL = args[3]
	}

	e.Author = &discordgo.MessageEmbedAuthor{
		Name:         name,
		IconURL:      iconURL,
		URL:          URL,
		ProxyIconURL: proxyURL,
	}
	return e
}

// SetURL sets the URL of the Embed. Returns the modified Embed.
func (e *Embed) SetURL(url string) *Embed {
	e.URL = url
	return e
}

// SetColor sets the border color of the Embed. Returns the modified
// Embed.
func (e *Embed) SetColor(color int) *Embed {
	e.Color = color
	return e
}

// SetAllFieldsInline sets all fields in the Embed to be
// inline. Returns the modified Embed
func (e *Embed) SetAllFieldsInline() *Embed {
	for _, v := range e.Fields {
		v.Inline = true
	}
	return e
}
