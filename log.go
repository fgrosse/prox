package prox

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

func logger(out *processOutput, debug bool) *zap.Logger {
	lvl := zap.WarnLevel
	if debug {
		lvl = zap.DebugLevel
	}

	enc := newLogEncoder()
	core := zapcore.NewCore(enc, zapcore.AddSync(out), lvl)

	return zap.New(core)
}

type logEncoder struct {
	zapcore.Encoder
	pool buffer.Pool
}

func newLogEncoder() zapcore.Encoder {
	// We want to omit the regular fields from the JSON encoder so we basically
	// leave all those *Key fields on their zero values.
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.LevelKey = ""
	cfg.TimeKey = ""
	cfg.NameKey = ""
	cfg.CallerKey = ""
	cfg.MessageKey = ""

	return logEncoder{
		Encoder: zapcore.NewJSONEncoder(cfg),
		pool:    buffer.NewPool(),
	}
}

func (c logEncoder) Clone() zapcore.Encoder {
	return logEncoder{
		Encoder: c.Encoder.Clone(),
		pool:    c.pool,
	}
}

func (c logEncoder) EncodeEntry(ent zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := c.pool.Get()

	line.AppendString(ent.Message)
	if len(fields) == 0 {
		return line, nil
	}

	// add all extra fields as JSON
	line.AppendString("\t")
	jsonEnc := c.Encoder.Clone()
	buf, err := jsonEnc.EncodeEntry(ent, fields)
	if err != nil {
		return nil, err
	}

	defer buf.Free()
	line.AppendString(buf.String())

	return line, nil
}
