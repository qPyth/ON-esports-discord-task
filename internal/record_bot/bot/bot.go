package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/qPyth/ON-esports-discord-task/internal/record_bot/handlers"
	"github.com/qPyth/ON-esports-discord-task/internal/record_bot/types"
	"github.com/qPyth/ON-esports-discord-task/internal/record_bot/usecases"
	"github.com/qPyth/ON-esports-discord-task/pkg"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func Start(token string, recordTime time.Duration, maxAttempts uint, awsCfg pkg.AWSConfig) {
	//init logger
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	awsS3 := pkg.NewAwsS3(awsCfg)

	//init usecases
	ucs := usecases.New(log, awsS3, types.Config{
		RecordTime:  recordTime,
		MaxAttempts: maxAttempts,
	})

	//init handlers
	h := handlers.New(awsS3, ucs.Recorder, log)

	//init bot session
	bot, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Error("new session creating error", "err", err.Error())
		return
	}

	// identify intents for bot to skip unnecessary events
	bot.Identify.Intents = discordgo.IntentDirectMessages | discordgo.IntentsGuilds | discordgo.IntentsGuildVoiceStates

	//add handlers to bot events
	bot.AddHandler(h.ChannelCreateHandler)
	bot.AddHandler(h.DirectMessage)

	// opening connection to listen events
	err = bot.Open()
	if err != nil {
		slog.Error("opening connection error", "err", err.Error())
		return
	}
	slog.Info("RecordBot is now running")

	// gracefully shutdown
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	bot.Close()
}
