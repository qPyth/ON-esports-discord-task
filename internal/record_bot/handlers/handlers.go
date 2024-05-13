package handlers

import (
	"errors"
	"github.com/bwmarrin/discordgo"
	"log/slog"
	"path"
	"strings"
)

type Recorder interface {
	Record(s *discordgo.Session, guildID, chanID string) error
}

type StorageLinkGenerator interface {
	GenerateLinks(folder string) ([]string, error)
}

var ErrServer = errors.New("something went wrong, try again later")

type Handler struct {
	sl       StorageLinkGenerator
	recorder Recorder
	log      *slog.Logger
}

func New(sl StorageLinkGenerator, recorder Recorder, log *slog.Logger) *Handler {
	return &Handler{sl: sl, recorder: recorder, log: log}
}

func (h Handler) ChannelCreateHandler(s *discordgo.Session, c *discordgo.ChannelCreate) {
	if c.Type != discordgo.ChannelTypeGuildVoice {
		return
	}

	h.log.Info("New channel created", "name", c.Name)

	err := h.recorder.Record(s, c.GuildID, c.ID)
	if err != nil {
		h.log.Warn("voice recording error", "err", err.Error())
		return
	}
}

func (h Handler) DirectMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, recordsCommand) {
		chanID := strings.Fields(m.Content)[1]

		links, err := h.sl.GenerateLinks(path.Join("records", chanID))
		if err != nil {
			_, err = s.ChannelMessageSend(m.ChannelID, ErrServer.Error())
			if err != nil {
				h.log.Error("message send error", "err", err.Error())
				return
			}
			h.log.Error("generate link error", "err", err.Error())
			return
		}

		var builder strings.Builder

		switch len(links) {
		case 0:
			builder.WriteString("there are no records for this channel")
		default:
			builder.WriteString("Links are only available for 15 minutes\n")

			for i, s2 := range links {
				builder.WriteString(s2)
				if i != len(links)-1 {
					builder.WriteString("\n")
				}
			}
		}

		_, err = s.ChannelMessageSend(m.ChannelID, builder.String())
		if err != nil {
			h.log.Error("message send error", "err", err.Error())
			return
		}
	}
}
