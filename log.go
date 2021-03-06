package prox

import (
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a *zap.Logger that emits its log messages through the given
// io.Writer. If debug is true the log level will also include debug and info
// messages. Otherwise only warning and errors will be logged.
func NewLogger(w io.Writer, debug bool) *zap.Logger {
	lvl := zap.WarnLevel
	if debug {
		lvl = zap.DebugLevel
	}

	enc := newLogEncoder()
	core := zapcore.NewCore(enc, zapcore.AddSync(w), lvl)

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

	line.AppendString("[")
	line.AppendString(ent.Level.CapitalString())
	line.AppendString("] ")

	line.AppendString(ent.Message)

	// add all extra fields as JSON
	jsonEnc := c.Encoder.Clone()
	buf, err := jsonEnc.EncodeEntry(ent, fields)
	if err != nil {
		return nil, err
	}

	defer buf.Free()
	if s := strings.TrimSpace(buf.String()); s != "{}" {
		line.AppendString("\t")
		line.AppendString(s)
	}

	line.AppendString("\n")
	return line, nil
}
