package prox

import (
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// logger creates a zap logger that emits its log messages through the given
// processOutput. If debug is true the log level will also include debug
// messages. Otherwise only warning and errors will be logged.
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
	cfg := zap.NewDevelopmentEncoderConfig()
	// We want to omit a couple of fields from the JSON encoder so we set the
	// corresponding fields to the empty string.
	cfg.LevelKey = ""
	cfg.TimeKey = ""
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
