package main

import (
	"github.com/qPyth/ON-esports-discord-task/config"
	"github.com/qPyth/ON-esports-discord-task/internal/record_bot/bot"
	"github.com/qPyth/ON-esports-discord-task/pkg"
	v "github.com/spf13/viper"
)

func main() {
	config.MustLoad()

	bot.Start(v.GetString("API_TOKEN"), v.GetDuration("RECORD_TIME"), v.GetUint("MAX_ATTEMPTS"), pkg.AWSConfig{
		Region:     v.GetString("REGION"),
		BucketName: v.GetString("BUCKET_NAME"),
	})
}
