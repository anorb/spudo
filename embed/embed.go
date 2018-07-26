package embed

import (
	"github.com/bwmarrin/discordgo"
	"gitlab.com/anorb/spudo/utils"
)

// Embed is a wrapper around *discordgo.MessageEmbed
type Embed struct {
	*discordgo.MessageEmbed
}

// Convert takes in a utils.Embed and changes it into a  discordgo.MessageEmbed
func Convert(old *utils.Embed) *Embed {
	new := newEmbed().setTitle(old.Title).setDescription(old.Description).setURL(old.URL).setColor(old.Color)

	if old.Footer != nil {
		new.setFooter(old.Footer.Text, old.Footer.IconURL, old.Footer.ProxyIconURL)
	}
	if old.Image != nil {
		new.setImage(old.Image.URL, old.Image.ProxyURL)
	}
	if old.Thumbnail != nil {
		new.setThumbnail(old.Thumbnail.URL, old.Thumbnail.ProxyURL)
	}
	if old.Author != nil {
		new.setAuthor(old.Author.Name, old.Author.IconURL, old.Author.URL, old.Author.ProxyIconURL)
	}
	for _, v := range old.Fields {
		new.addField(v.Name, v.Value, v.Inline)
	}
	return new
}

func newEmbed() *Embed {
	return &Embed{&discordgo.MessageEmbed{}}
}

func (e *Embed) setTitle(title string) *Embed {
	e.Title = title
	return e
}

func (e *Embed) setDescription(description string) *Embed {
	e.Description = description
	return e
}

func (e *Embed) addField(name, value string, inline bool) *Embed {
	e.Fields = append(e.Fields, &discordgo.MessageEmbedField{
		Name:   name,
		Value:  value,
		Inline: inline,
	})
	return e
}

func (e *Embed) setFooter(args ...string) *Embed {
	var text string
	var iconURL string
	var proxyURL string

	if len(args) == 0 {
		return e
	}
	if len(args) > 0 {
		text = args[0]
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

func (e *Embed) setImage(args ...string) *Embed {
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

func (e *Embed) setThumbnail(args ...string) *Embed {
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

func (e *Embed) setAuthor(args ...string) *Embed {
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

func (e *Embed) setURL(URL string) *Embed {
	e.URL = URL
	return e
}

func (e *Embed) setColor(color int) *Embed {
	e.Color = color
	return e
}
