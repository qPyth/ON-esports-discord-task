package usecases

import (
	"github.com/qPyth/ON-esports-discord-task/internal/record_bot/types"
	"log/slog"
)

type UseCases struct {
	Recorder *RecordUC
}

func New(logger *slog.Logger, cs CloudSaver, cfg types.Config) *UseCases {
	return &UseCases{
		Recorder: NewRecordUC(cs, logger, cfg.MaxAttempts, cfg.RecordTime),
	}
}
