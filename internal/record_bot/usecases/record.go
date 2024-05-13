package usecases

import (
	"context"
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/pion/webrtc/v3/pkg/media/oggwriter"
	"io"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type CloudSaver interface {
	Save(data []byte, name string) error
}

type RecordUC struct {
	cs             CloudSaver
	log            *slog.Logger
	maxAttempts    uint
	recordDuration time.Duration
}

func NewRecordUC(cs CloudSaver, log *slog.Logger, maxAttempts uint, recordDuration time.Duration) *RecordUC {
	return &RecordUC{cs: cs, log: log, maxAttempts: maxAttempts, recordDuration: recordDuration}
}

func (r *RecordUC) Record(s *discordgo.Session, guildID, chanID string) error {
	vc, err := r.joinToChannelWithAttempts(s, guildID, chanID)
	if err != nil {
		r.log.Error("channel join error", "err", err)
		return err
	}
	defer vc.Disconnect()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//if timer not equal 0, create a timer in new goroutine
	if r.recordDuration > 0 {
		go func() {
			select {
			case <-time.After(r.recordDuration):
				cancel()
			case <-ctx.Done():
				return
			}
		}()
	}

	//The goroutine monitors for context cancellations from the timer or from other parts
	//of the application and closes all connections.
	//It also checks for the presence of people on the server every second,
	//if the room is empty it also ends recording and closes all connections
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		var userEntered bool

		for {
			select {
			case <-ctx.Done():
				close(vc.OpusRecv)
				r.log.Info("record stopped, time is over")
				return
			case <-ticker.C:
				exist, err := isAnyoneInVoiceChannel(s, guildID, chanID)
				if err != nil {
					close(vc.OpusRecv)
					r.log.Error("check users in channel error", "err", err)
					cancel()
					return
				}
				if !userEntered && exist {
					userEntered = true
				}
				if !exist && userEntered {
					close(vc.OpusRecv)
					r.log.Info("record stopped, channel is empty")
					cancel()
					return
				}
			}
		}
	}()

	r.log.Info("Recording started", "channel", chanID)

	//creating a directory to store voice files of the current call
	fld := path.Join("records", chanID)
	err = os.MkdirAll(fld, 0755)
	if err != nil {
		return err
	}

	//voice recording
	handleVoice(vc.OpusRecv, fld)

	go Save(r.cs, fld)
	return nil
}

func (r *RecordUC) joinToChannelWithAttempts(s *discordgo.Session, guildID, chanID string) (vc *discordgo.VoiceConnection, err error) {
	if chanID == "" {
		return nil, fmt.Errorf("chanID is empty")
	}

	if r.maxAttempts == 0 {
		r.maxAttempts = 5
	}
	for i := 0; i < int(r.maxAttempts); i++ {
		r.log.Info("Attempt to connect to voice channel", "attempt", i+1, "channelID", chanID)
		vc, err = s.ChannelVoiceJoin(guildID, chanID, true, false)
		if err != nil {
			r.log.Warn("Failed to join voice channel", "err", err.Error())
			time.Sleep(time.Second * time.Duration(i+1))
			continue
		}
		if vc.Ready {
			r.log.Info("Connected to voice channel successfully", "attempt", i+1, "channelID", chanID)
			break
		}
	}
	return
}

func isAnyoneInVoiceChannel(s *discordgo.Session, guildID string, channelID string) (bool, error) {
	guild, err := s.State.Guild(guildID)
	if err != nil {
		guild, err = s.Guild(guildID)
		if err != nil {
			return false, err
		}
	}

	for _, vs := range guild.VoiceStates {
		if vs.ChannelID == channelID {
			user, err := s.User(vs.UserID)
			if err != nil {
				continue
			}
			if !user.Bot {
				return true, nil
			}
		}
	}

	return false, nil
}

func handleVoice(c chan *discordgo.Packet, fp string) {

	files := make(map[uint32]media.Writer)
	for p := range c {
		file, ok := files[p.SSRC]
		if !ok {
			var err error
			filename := path.Join(fp, fmt.Sprintf("%d.ogg", p.SSRC))
			file, err = oggwriter.New(filename, 48000, 2)
			if err != nil {
				fmt.Printf("failed to create file %d.ogg, giving up on recording: %v\n", p.SSRC, err)
				return
			}
			files[p.SSRC] = file
		}
		rtpPacket := createPionRTPPacket(p)
		err := file.WriteRTP(rtpPacket)
		if err != nil {
			fmt.Printf("failed to write to file %d.ogg, giving up on recording: %v\n", p.SSRC, err)
		}
	}

	for _, f := range files {
		f.Close()
	}
}

func Save(saver CloudSaver, fld string) {
	dir, err := os.ReadDir(fld)
	if err != nil {
		slog.Error("open dir error", "err", err.Error())
		return
	}

	var wg sync.WaitGroup
	for _, entry := range dir {
		if !entry.IsDir() {
			if strings.HasSuffix(entry.Name(), ".ogg") {
				wg.Add(1)
				go func(name string) {
					defer wg.Done()
					filepath := path.Join(fld, name)

					file, err := os.Open(filepath)
					if err != nil {
						slog.Error("error open file", "err", err.Error())
						return
					}

					data, err := io.ReadAll(file)
					if err != nil {
						slog.Error("error open file", "err", err.Error())
						return
					}

					err = saver.Save(data, path.Join(filepath, name))
					if err != nil {
						slog.Error("save file to cloud error", "err", err.Error())
						return
					}
					_ = file.Close()
					err = os.Remove(filepath)
					if err != nil {
						slog.Error("file remove error", "err", err.Error())
						return
					}
					slog.Info("record sent successfully", "file", path.Base(filepath))

				}(entry.Name())
			}
		}
	}
	wg.Wait()
	err = removeEmptyDir(fld)
	if err != nil {
		if errors.Is(err, errDirNotEmpty) {
			slog.Info("some files were not sent to cloud storage")
			return
		}
		slog.Error("remove dir error", "err", err.Error())
	}
}

func createPionRTPPacket(p *discordgo.Packet) *rtp.Packet {
	return &rtp.Packet{
		Header: rtp.Header{
			Version:        2,
			PayloadType:    0x78,
			SequenceNumber: p.Sequence,
			Timestamp:      p.Timestamp,
			SSRC:           p.SSRC,
		},
		Payload: p.Opus,
	}
}
