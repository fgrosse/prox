package prox

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func logger(debug bool) *zap.Logger {
	conf := zap.NewDevelopmentConfig()
	conf.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("15:04:05"))
	}
	conf.EncoderConfig.EncodeCaller = nil
	conf.Level = zap.NewAtomicLevelAt(zap.WarnLevel)

	if debug {
		conf.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	logger, err := conf.Build()
	if err != nil {
		panic(err)
	}

	return logger.Named("prox")
}
