package types

import "time"

type Config struct {
	RecordTime  time.Duration
	MaxAttempts uint
}
